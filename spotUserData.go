package bnnapi

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

type spotUserDataBranch struct {
	account            spotAccountBranch
	cancel             *context.CancelFunc
	httpUpdateInterval int
	errs               chan error
	trades             userTradesBranch
}

type userTradesBranch struct {
	sync.RWMutex
	data []TradeData
}

type spotAccountBranch struct {
	sync.RWMutex
	Data *SpotAccountResponse
}

type TradeData struct {
	Symbol    string
	Side      string
	Oid       string
	IsMaker   bool
	OrderType string
	Price     decimal.Decimal
	Qty       decimal.Decimal
	Fee       decimal.Decimal
	FeeAsset  string
	TimeStamp time.Time
}

func (u *Client) CloseSpotUserData() {
	(*u.spotUser.cancel)()
}

// default is 60 sec
func (c *Client) SetSpotHttpUpdateInterval(input int) {
	c.spotUser.httpUpdateInterval = input
}

func (c *Client) GetSpotAccountData() (*SpotAccountResponse, error) {
	c.spotUser.account.RLock()
	defer c.spotUser.account.RUnlock()
	return c.spotUser.account.Data, c.spotUser.readerrs()
}

func (c *Client) ReadSpotUserTrade() []TradeData {
	c.spotUser.trades.Lock()
	defer c.spotUser.trades.Unlock()
	trades := c.spotUser.trades.data
	c.spotUser.trades.data = []TradeData{}
	return trades
}

func (c *Client) InitSpotPrivateChannel(logger *log.Logger) {
	c.spotLocalUserData(logger)
}

// spot, perp, isomargin, margin
func (c *Client) GetListenKeyHub(product, symbol string) (*ListenKeyResponse, error) {
	switch product {
	case "spot":
		res, err := c.GetSpotListenKey()
		if err != nil {
			return nil, err
		}
		return res, nil
	case "margin":
		res, err := c.GetMarginListenKey()
		if err != nil {
			return nil, err
		}
		return res, nil
	case "isomargin":
		res, err := c.GetIsolatedMarginListenKey(symbol)
		if err != nil {
			return nil, err
		}
		return res, nil
	case "perp":
		res, err := c.GetPerpListenKey()
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	return nil, errors.New("unsupported product")
}

// spot, perp, isomargin, margin
func (c *Client) PutListenKeyHub(product, listenKey string) error {
	switch product {
	case "spot":
		err := c.PutSpotListenKey(listenKey)
		if err != nil {
			return err
		}
		return nil
	case "margin":
		err := c.PutMarginListenKey(listenKey)
		if err != nil {
			return err
		}
		return nil
	case "isomargin":
		err := c.PutIsoMarginListenKey(listenKey)
		if err != nil {
			return err
		}
		return nil
	case "perp":
		err := c.PutPerpListenKey(listenKey)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("unsupported product")
}

// internal funcs ------------------------------------------------

// default errs cap 5, trades cap 100
func (c *Client) spotLocalUserData(logger *log.Logger) {
	var u spotUserDataBranch
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = &cancel
	u.httpUpdateInterval = 60
	u.initialChannels()
	userData := make(chan map[string]interface{}, 100)
	// stream user data
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				res, err := c.GetListenKeyHub("spot", "") // delete listen key
				if err != nil {
					logger.Println("retry listen key for user data stream in 5 sec..")
					time.Sleep(time.Second * 5)
					continue
				}
				if err := c.spotUserData(ctx, res.ListenKey, logger, &userData); err == nil {
					return
				}
				time.Sleep(time.Second)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := u.maintainUserData(ctx, c, &userData); err == nil {
					return
				} else {
					logger.Warningf("Refreshing private channel with err: %s.\n", err.Error())
				}
			}
		}
	}()
	// wait for connecting
	c.spotUser = &u
	time.Sleep(time.Second * 5)
}

func (u *spotUserDataBranch) getAccountSnapShot(client *Client) error {
	u.account.Lock()
	defer u.account.Unlock()
	res, err := client.SpotAccount()
	if err != nil {
		return err
	}
	u.account.Data = res
	return nil
}

func (u *spotUserDataBranch) maintainUserData(
	ctx context.Context,
	client *Client,
	userData *chan map[string]interface{},
) error {
	// get the first snapshot to initial data struct
	if err := u.getAccountSnapShot(client); err != nil {
		return err
	}
	// update snapshot with steady interval
	go func() {
		snap := time.NewTicker(time.Second * time.Duration(u.httpUpdateInterval))
		defer snap.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-snap.C:
				if err := u.getAccountSnapShot(client); err != nil {
					u.insertErr(err)
				}
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			close(u.errs)
			return nil
		default:
			message := <-(*userData)
			event, ok := message["e"].(string)
			if !ok {
				continue
			}
			switch event {
			case "outboundAccountPosition":
				if data, ok := message["a"].(map[string]interface{}); !ok {
					continue
				} else {
					u.updateAccountData(&data)
				}
			case "executionReport":
				if event, ok := message["x"].(string); ok {
					switch event {
					case "TRADE":
						u.handleTrade(&message)
					default:
						// order update in the future
					}
				}
			}
		}
	}
}

func (c *Client) spotUserData(ctx context.Context, listenKey string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w wS
	var duration time.Duration = 1810
	w.Logger = logger
	w.OnErr = false
	var buffer bytes.Buffer
	innerErr := make(chan error, 1)
	buffer.WriteString("wss://stream.binance.com:9443/ws/")
	buffer.WriteString(listenKey)
	url := buffer.String()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	log.Println("Connected:", url)
	w.Conn = conn
	defer w.Conn.Close()
	if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
		return err
	}
	w.Conn.SetPingHandler(nil)
	go func() {
		putKey := time.NewTicker(time.Minute * 30)
		defer putKey.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-innerErr:
				return
			case <-putKey.C:
				if err := c.PutListenKeyHub("spot", listenKey); err != nil {
					// time out in 1 sec
					w.Conn.SetReadDeadline(time.Now().Add(time.Second))
				}
				w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration))
			}
		}
	}()
	read := time.NewTicker(time.Millisecond * 100)
	defer read.Stop()
	for {
		select {
		case <-ctx.Done():
			w.outBinanceErr()
			message := "Binance spot private channel closed..."
			return errors.New(message)
		case <-read.C:
			if w.Conn == nil {
				w.outBinanceErr()
				message := "Binance spot private channel reconnect..."
				innerErr <- errors.New("restart")
				return errors.New(message)
			}
			_, buf, err := w.Conn.ReadMessage()
			if err != nil {
				w.outBinanceErr()
				innerErr <- errors.New("restart")
				return err
			}
			res, err1 := decodingMap(buf, logger)
			if err1 != nil {
				w.outBinanceErr()
				innerErr <- errors.New("restart")
				return err1
			}
			// check event time first
			c.spotUser.handleUserData(&res, mainCh)
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				innerErr <- errors.New("restart")
				return err
			}
		}
	}
}

func (u *spotUserDataBranch) handleUserData(res *map[string]interface{}, mainCh *chan map[string]interface{}) {
	if eventTimeUnix, ok := (*res)["E"].(float64); ok {
		eventTime := formatingTimeStamp(eventTimeUnix)
		if time.Now().After(eventTime.Add(time.Minute * 60)) {
			return
		}
		// insert to chan
		*mainCh <- *res
	}
}

func (u *spotUserDataBranch) initialChannels() {
	// 5 err is allowed
	u.errs = make(chan error, 5)
}

func (u *spotUserDataBranch) insertErr(input error) {
	if len(u.errs) == cap(u.errs) {
		<-u.errs
	}
	u.errs <- input
}

func (u *spotUserDataBranch) insertTrade(input *TradeData) {
	u.trades.Lock()
	defer u.trades.Unlock()
	u.trades.data = append(u.trades.data, *input)
}

func (u *spotUserDataBranch) readerrs() error {
	var buffer bytes.Buffer
	for {
		select {
		case err, ok := <-u.errs:
			if ok {
				buffer.WriteString(err.Error())
				buffer.WriteString(", ")
			} else {
				buffer.WriteString("errs chan already closed, ")
			}
		default:
			if buffer.Cap() == 0 {
				return nil
			}
			return errors.New(buffer.String())
		}
	}
}

// default fee asset is USDT
func (u *spotUserDataBranch) handleTrade(res *map[string]interface{}) {
	data := TradeData{}
	if symbol, ok := (*res)["s"].(string); ok {
		data.Symbol = symbol
	} else {
		return
	}
	if side, ok := (*res)["S"].(string); ok {
		data.Side = strings.ToLower(side)
	} else {
		return
	}
	if qty, ok := (*res)["l"].(string); ok {
		data.Qty, _ = decimal.NewFromString(qty)
	} else {
		return
	}
	if price, ok := (*res)["L"].(string); ok {
		data.Price, _ = decimal.NewFromString(price)
	} else {
		return
	}
	if oid, ok := (*res)["i"].(float64); ok {
		data.Oid = decimal.NewFromFloat(oid).String()
	} else {
		return
	}
	if execType, ok := (*res)["m"].(bool); ok {
		data.IsMaker = execType
	}
	if fee, ok := (*res)["n"].(string); ok {
		data.Fee, _ = decimal.NewFromString(fee)
	}
	if st, ok := (*res)["T"].(float64); ok {
		stamp := formatingTimeStamp(st)
		data.TimeStamp = stamp
	}
	if orderType, ok := (*res)["o"].(string); ok {
		data.OrderType = orderType
	}
	if feeAsset, ok := (*res)["N"].(string); ok {
		data.FeeAsset = feeAsset
	}
	u.insertTrade(&data)
}

func (u *spotUserDataBranch) updateAccountData(message *map[string]interface{}) {
	array, ok := (*message)["B"].([]interface{})
	if !ok {
		return
	}
	u.account.Lock()
	defer u.account.Unlock()
	for _, item := range array {
		data, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		asset, oka := data["a"].(string)
		if !oka {
			continue
		}
		free, okf := data["f"].(string)
		if !okf {
			continue
		}
		lock, okl := data["l"].(string)
		if !okl {
			continue
		}
		for idx, bal := range u.account.Data.Balances {
			if bal.Asset == asset {
				u.account.Data.Balances[idx].Free = free
				u.account.Data.Balances[idx].Locked = lock
				return
			}
		}
	}
}
