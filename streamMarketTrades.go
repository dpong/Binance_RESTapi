package bnnapi

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

type StreamMarketTradesBranch struct {
	cancel *context.CancelFunc
	conn   *websocket.Conn
	market string

	tradeChan    chan BnnTradeData
	tradesBranch struct {
		Trades []BnnTradeData
		sync.Mutex
	}
	logger *logrus.Logger
}

type BnnTradeData struct {
	Event     string `json:"e"`
	EventTime int    `json:"E"`
	Symbol    string `json:"s"`
	TradeId   int    `json:"t"`
	Price     string `json:"p"`
	Qty       string `json:"q"`
	BidId     int    `json:"b"`
	AskId     int    `json:"a"`
	Timestamp int    `json:"T"`
	Maker     bool   `json:"m"`
	M         bool   `json:"M"`
}

func SwapTradeStream(symbol string, logger *logrus.Logger) *StreamMarketTradesBranch {
	Usymbol := strings.ToUpper(symbol)
	return tradeStream(Usymbol, logger, "swap")
}

func SpotTradeStream(symbol string, logger *logrus.Logger) *StreamMarketTradesBranch {
	Usymbol := strings.ToUpper(symbol)
	return tradeStream(Usymbol, logger, "spot")
}

func (o *StreamMarketTradesBranch) GetTrades() []BnnTradeData {
	o.tradesBranch.Lock()
	defer o.tradesBranch.Unlock()
	trades := o.tradesBranch.Trades
	o.tradesBranch.Trades = []BnnTradeData{}
	return trades
}

func (o *StreamMarketTradesBranch) Close() {
	(*o.cancel)()
	o.tradesBranch.Lock()
	defer o.tradesBranch.Unlock()
	o.tradesBranch.Trades = []BnnTradeData{}
}

func tradeStream(symbol string, logger *logrus.Logger, product string) *StreamMarketTradesBranch {
	o := new(StreamMarketTradesBranch)
	ctx, cancel := context.WithCancel(context.Background())
	o.cancel = &cancel
	o.market = symbol
	o.tradeChan = make(chan BnnTradeData, 100)
	o.logger = logger
	go o.maintainSession(ctx, product, symbol)
	go o.listen(ctx)
	return o
}

func (o *StreamMarketTradesBranch) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case trade := <-o.tradeChan:
			o.appendNewTrade(&trade)
		default:
			time.Sleep(time.Second)
		}
	}
}

func (o *StreamMarketTradesBranch) appendNewTrade(new *BnnTradeData) {
	o.tradesBranch.Lock()
	defer o.tradesBranch.Unlock()
	o.tradesBranch.Trades = append(o.tradesBranch.Trades, *new)
}

func (o *StreamMarketTradesBranch) maintainSession(ctx context.Context, product, symbol string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := o.maintain(ctx, product, symbol); err == nil {
				return
			} else {
				o.logger.Warningf("reconnect Binance %s %s trade stream with err: %s\n", symbol, product, err.Error())
			}
		}
	}
}

func (o *StreamMarketTradesBranch) maintain(ctx context.Context, product string, symbol string) error {
	var duration time.Duration = 30
	var buffer bytes.Buffer
	switch product {
	case "spot":
		buffer.WriteString("wss://stream.binance.com:9443/ws/")
	case "swap":
		buffer.WriteString("wss://fstream.binance.com/ws/")
	}
	buffer.WriteString(strings.ToLower(symbol))
	buffer.WriteString("@trade")
	conn, _, err := websocket.DefaultDialer.Dial(buffer.String(), nil)
	if err != nil {
		return err
	}
	o.conn = conn
	defer o.conn.Close()
	if err := o.conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
		return err
	}
	o.conn.SetPingHandler(nil)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			_, msg, err := o.conn.ReadMessage()
			if err != nil {
				return err
			}
			if err := o.handleBnnTradeSocketMsg(msg); err != nil {
				return err
			}
			if err := o.conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
		} // end select
	} // end for
}

func (o *StreamMarketTradesBranch) handleBnnTradeSocketMsg(msg []byte) error {
	var data BnnTradeData
	err := json.Unmarshal(msg, &data)
	if err != nil {
		return errors.New("fail to unmarshal message")
	}
	// distribute the msg
	switch data.Event {
	case "subscribed":
		//fmt.Println("websocket subscribed")
	case "trade":
		o.tradeChan <- data
	default:
		// pass
	}
	return nil
}
