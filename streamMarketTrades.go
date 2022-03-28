package bnnapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

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

	lastUpdatedTimestampBranch struct {
		timestamp int64
		mux       sync.RWMutex
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
	return tradeStream(symbol, logger, "swap")
}

func SpotTradeStream(symbol string, logger *logrus.Logger) *StreamMarketTradesBranch {
	return tradeStream(symbol, logger, "spot")
}

func (o *StreamMarketTradesBranch) GetTrades() []BnnTradeData {
	o.tradesBranch.Lock()
	defer o.tradesBranch.Unlock()
	trades := o.tradesBranch.Trades
	o.tradesBranch.Trades = []BnnTradeData{}
	return trades
}

func tradeStream(symbol string, logger *logrus.Logger, product string) *StreamMarketTradesBranch {
	o := new(StreamMarketTradesBranch)
	ctx, cancel := context.WithCancel(context.Background())
	o.cancel = &cancel
	o.market = symbol
	o.tradeChan = make(chan BnnTradeData, 100)
	o.logger = logger
	go o.maintain(ctx, product, symbol)
	go o.listen(ctx)

	time.Sleep(1 * time.Second)
	go o.pingIt(ctx)

	return o
}

func (o *StreamMarketTradesBranch) pingIt(ctx context.Context) {
	ping := time.NewTicker(time.Second * 60)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			message := []byte("pong")
			o.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
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

func (o StreamMarketTradesBranch) maintainSession(ctx context.Context, product, symbol string) {

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
			var msgMap map[string]interface{}
			err = json.Unmarshal(msg, &msgMap)
			if err != nil {
				return err
			}
			errh := o.handleBnnTradeSocketMsg(msg)
			if errh != nil {
				return errh
			}
		} // end select
	} // end for
}

func (o *StreamMarketTradesBranch) handleBnnTradeSocketMsg(msg []byte) error {
	var msgMap map[string]interface{}
	err := json.Unmarshal(msg, &msgMap)
	if err != nil {
		log.Println(err)
		return errors.New("fail to unmarshal message")
	}

	event, ok := msgMap["e"]
	if !ok {
		return errors.New("fail to obtain message")
	}

	// distribute the msg
	var err2 error
	switch event {
	case "subscribed":
		//fmt.Println("websocket subscribed")
	case "snapshot":
		//err2 = o.parseOrderbookSnapshotMsg(msgMap)
	case "trade":
		err2 = o.parseTradeUpdateMsg(msg)
	default:
		log.Println(msg)
	}
	if err2 != nil {
		fmt.Println(err2, "err2")
		return errors.New("fail to parse message")
	}
	return nil
}

func (o *StreamMarketTradesBranch) parseTradeUpdateMsg(msg []byte) error {
	var tradeData BnnTradeData
	json.Unmarshal(msg, &tradeData)

	// extract data
	if tradeData.Event != "trade" {
		return errors.New("wrong channel")
	}
	if tradeData.Symbol != o.market {
		return errors.New("wrong market")
	}
	o.tradeChan <- tradeData

	o.lastUpdatedTimestampBranch.mux.Lock()
	defer o.lastUpdatedTimestampBranch.mux.Unlock()
	o.lastUpdatedTimestampBranch.timestamp = int64(tradeData.Timestamp)
	return nil
}
