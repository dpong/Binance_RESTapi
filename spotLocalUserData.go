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

type SpotUserDataBranch struct {
	account            spotAccountBranch
	cancel             *context.CancelFunc
	httpUpdateInterval int
	errs               chan error
	trades             chan TradeData
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
	TimeStamp time.Time
}

func (u *Client) CloseSpotUserData() {
	(*u.spotUser.cancel)()
}

// default is 60 sec
func (u *SpotUserDataBranch) SetHttpUpdateInterval(input int) {
	u.httpUpdateInterval = input
}

func (u *SpotUserDataBranch) AccountData() (*SpotAccountResponse, error) {
	u.account.RLock()
	defer u.account.RUnlock()
	return u.account.Data, u.readerrs()
}

func (u *SpotUserDataBranch) ReadTrade() (TradeData, error) {
	if len(u.trades) == 0 {
		return TradeData{}, errors.New("trade channel is empty")
	}
	if data, ok := <-u.trades; ok {
		return data, nil
	}
	return TradeData{}, errors.New("trade channel is closed.")
}

func (c *Client) InitSpotUserData(logger *log.Logger) {
	c.spotLocalUserData(logger)
}

func (u *SpotUserDataBranch) SpotAccount() *SpotAccountResponse {
	u.account.RLock()
	defer u.account.RUnlock()
	return u.account.Data
}

// spot, swap, isomargin, margin
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
	case "swap":
		res, err := c.GetSwapListenKey()
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	return nil, errors.New("unsupported product")
}

// spot, swap, isomargin, margin
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
	case "swap":
		err := c.PutSwapListenKey(listenKey)
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
	var u SpotUserDataBranch
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
					log.Println("retry listen key for user data stream in 5 sec..")
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
					logger.Warningf("Refreshing apx local user data with err: %s.\n", err.Error())
				}
			}
		}
	}()
	// wait for connecting
	time.Sleep(time.Second * 5)
	c.spotUser = &u
}

func (u *SpotUserDataBranch) getAccountSnapShot(client *Client) error {
	u.account.Lock()
	defer u.account.Unlock()
	res, err := client.SpotAccount()
	if err != nil {
		return err
	}
	u.account.Data = res
	return nil
}

func (u *SpotUserDataBranch) maintainUserData(
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
			default:
				time.Sleep(time.Second)
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			close(u.errs)
			close(u.trades)
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
			default:
				// pass
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
			default:
				time.Sleep(time.Second)
			}
		}
	}()
	read := time.NewTicker(time.Millisecond * 100)
	defer read.Stop()
	for {
		select {
		case <-ctx.Done():
			w.outBinanceErr()
			message := "Apx User Data closed..."
			log.Println(message)
			return errors.New(message)
		case <-read.C:
			if w.Conn == nil {
				w.outBinanceErr()
				message := "Apx User Data reconnect..."
				log.Println(message)
				innerErr <- errors.New("restart")
				return errors.New(message)
			}
			_, buf, err := w.Conn.ReadMessage()
			if err != nil {
				w.outBinanceErr()
				message := "Apx User Data reconnect..."
				log.Println(message)
				innerErr <- errors.New("restart")
				return errors.New(message)
			}
			res, err1 := decodingMap(buf, logger)
			if err1 != nil {
				w.outBinanceErr()
				message := "Apx User Data reconnect..."
				log.Println(message, err1)
				innerErr <- errors.New("restart")
				return err1
			}
			// check event time first
			handleUserData(&res, mainCh)
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				innerErr <- errors.New("restart")
				return err
			}
		default:
			time.Sleep(time.Millisecond * 50)
		}
	}
}

func handleUserData(res *map[string]interface{}, mainCh *chan map[string]interface{}) {
	if eventTimeUnix, ok := (*res)["E"].(float64); ok {
		eventTime := formatingTimeStamp(eventTimeUnix)
		if time.Now().After(eventTime.Add(time.Minute * 60)) {
			return
		}
		// insert to chan
		*mainCh <- *res
	}
}

func (u *SpotUserDataBranch) initialChannels() {
	// 5 err is allowed
	u.errs = make(chan error, 5)
	u.trades = make(chan TradeData, 200)
}

func (u *SpotUserDataBranch) insertErr(input error) {
	if len(u.errs) == cap(u.errs) {
		<-u.errs
	}
	u.errs <- input
}

func (u *SpotUserDataBranch) insertTrade(input *TradeData) {
	if len(u.trades) == cap(u.trades) {
		<-u.trades
	}
	u.trades <- *input
}

func (u *SpotUserDataBranch) readerrs() error {
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
func (u *SpotUserDataBranch) handleTrade(res *map[string]interface{}) {
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
	u.insertTrade(&data)
}

func (u *SpotUserDataBranch) updateAccountData(message *map[string]interface{}) {
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
