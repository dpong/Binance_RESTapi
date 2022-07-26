package bnnapi

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type perpUserDataBranch struct {
	account            perpAccountBranch
	cancel             *context.CancelFunc
	httpUpdateInterval int
	errs               chan error
	trades             userTradesBranch
}

type perpAccountBranch struct {
	sync.RWMutex
	Data *PerpAccountResponse
}

type perpBalanceBranch struct {
	sync.RWMutex
	Data []*PerpBalanceResponse
}

func (u *Client) ClosePerpUserData() {
	(*u.perpUser.cancel)()
	u.perpUser.trades.Lock()
	defer u.perpUser.trades.Unlock()
	u.perpUser.trades.data = []TradeData{}
}

// default is 60 sec
func (c *Client) SetPerpHttpUpdateInterval(input int) {
	c.perpUser.httpUpdateInterval = input
}

func (c *Client) GetPerpAccountData() (*PerpAccountResponse, error) {
	c.perpUser.account.RLock()
	defer c.perpUser.account.RUnlock()
	return c.perpUser.account.Data, c.perpUser.readerrs()
}

func (c *Client) ReadPerpUserTrade() []TradeData {
	c.perpUser.trades.RLock()
	defer c.perpUser.trades.RUnlock()
	trades := c.perpUser.trades.data
	c.perpUser.trades.data = []TradeData{}
	return trades
}

func (c *Client) InitPerpPrivateChannel(logger *log.Logger) {
	c.perpLocalUserData(logger)
}

// internal funcs ------------------------------------------------

// default errs cap 5, trades cap 100
func (c *Client) perpLocalUserData(logger *log.Logger) {
	var u perpUserDataBranch
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
				res, err := c.GetListenKeyHub("perp", "") // delete listen key
				if err != nil {
					logger.Println("retry listen key for user data stream in 5 sec..")
					time.Sleep(time.Second * 5)
					continue
				}
				if err := c.perpUserData(ctx, res.ListenKey, logger, &userData); err == nil {
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
					logger.Warningf("Refreshing binancne private channel with err: %s.\n", err.Error())
				}
			}
		}
	}()
	// wait for connecting
	c.perpUser = &u
	time.Sleep(time.Second * 5)
}

func (u *perpUserDataBranch) getAccountSnapShot(client *Client) error {
	u.account.Lock()
	defer u.account.Unlock()
	res, err := client.PerpAccount()
	if err != nil {
		return err
	}
	u.account.Data = res
	return nil
}

func (u *perpUserDataBranch) maintainUserData(
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
			return nil
		default:
			message := <-(*userData)
			event, ok := message["e"].(string)
			if !ok {
				continue
			}
			switch event {
			case "ACCOUNT_UPDATE":
				if data, ok := message["a"].(map[string]interface{}); !ok {
					continue
				} else {
					u.updateAccountData(&data)
				}
			case "ORDER_TRADE_UPDATE":
				if event, ok := message["o"].(map[string]interface{}); ok {
					// only handle trade now
					if status, ok := event["X"].(string); ok {
						if status == "FILLED" || status == "PARTIALLY_FILLED" {
							u.handleTrade(&event)
						}
					}
				}
			default:
				// pass
			}
		}
	}
}

func (c *Client) perpUserData(ctx context.Context, listenKey string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w wS
	var duration time.Duration = 1810
	w.Logger = logger
	w.OnErr = false
	var buffer bytes.Buffer
	innerErr := make(chan error, 1)
	buffer.WriteString("wss://fstream.binance.com/ws/")
	buffer.WriteString(listenKey)
	url := buffer.String()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	logger.Println("Connected:", url)
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
				if err := c.PutListenKeyHub("perp", listenKey); err != nil {
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
			message := "Binance perp private channel closed..."
			return errors.New(message)
		case <-read.C:
			if w.Conn == nil {
				w.outBinanceErr()
				message := "Binance perp private channel reconnect..."
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
			c.perpUser.handleUserData(&res, mainCh)
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				innerErr <- errors.New("restart")
				return err
			}
		}
	}
}

func (u *perpUserDataBranch) handleUserData(res *map[string]interface{}, mainCh *chan map[string]interface{}) {
	if eventTimeUnix, ok := (*res)["E"].(float64); ok {
		eventTime := time.UnixMilli(int64(eventTimeUnix))
		if time.Now().After(eventTime.Add(time.Minute * 5)) {
			return
		}
		// insert to chan
		*mainCh <- *res
	}
}

func (u *perpUserDataBranch) initialChannels() {
	// 5 err is allowed
	u.errs = make(chan error, 5)
}

func (u *perpUserDataBranch) insertErr(input error) {
	if len(u.errs) == cap(u.errs) {
		<-u.errs
	}
	u.errs <- input
}

func (u *perpUserDataBranch) insertTrade(input *TradeData) {
	u.trades.Lock()
	defer u.trades.Unlock()
	u.trades.data = append(u.trades.data, *input)
}

func (u *perpUserDataBranch) readerrs() error {
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
func (u *perpUserDataBranch) handleTrade(res *map[string]interface{}) {
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
	if feeAsset, ok := (*res)["N"].(string); ok {
		data.FeeAsset = feeAsset
	}
	if st, ok := (*res)["T"].(float64); ok {
		stamp := time.UnixMilli(int64(st))
		data.TimeStamp = stamp
	}
	if orderType, ok := (*res)["o"].(string); ok {
		data.OrderType = orderType
	}
	u.insertTrade(&data)
}

func (u *perpUserDataBranch) updateAccountData(message *map[string]interface{}) {
	u.account.Lock()
	defer u.account.Unlock()
	if balances, ok := (*message)["B"].([]interface{}); ok {
		for _, item := range balances {
			data := item.(map[string]interface{})
			for idx, asset := range u.account.Data.Assets {
				if asset.Asset == data["a"] {
					u.account.Data.Assets[idx].WalletBalance = data["wb"].(string)
					u.account.Data.Assets[idx].CrossWalletBalance = data["cw"].(string)
				}
			}
		}
	}
	if positions, ok := (*message)["P"].([]interface{}); ok {
		for _, item := range positions {
			data := item.(map[string]interface{})
			for idx, position := range u.account.Data.Positions {
				if position.Symbol == data["s"] {
					u.account.Data.Positions[idx].PositionAmt = data["pa"].(string)
					u.account.Data.Positions[idx].EntryPrice = data["ep"].(string)
					u.account.Data.Positions[idx].UnrealizedProfit = data["up"].(string)
				}
			}
		}
	}
}
