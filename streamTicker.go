package bnnapi

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

const NullPrice = "null"

type StreamTickerBranch struct {
	bid    tobBranch
	ask    tobBranch
	cancel *context.CancelFunc
	reCh   chan error
	socket wS
}

type tobBranch struct {
	mux       sync.RWMutex
	price     string
	qty       string
	timestamp time.Time
}

func PerpStreamTicker(symbol string, logger *log.Logger) *StreamTickerBranch {
	return localStreamTicker("perp", symbol, logger)
}

func SpotStreamTicker(symbol string, logger *log.Logger) *StreamTickerBranch {
	return localStreamTicker("spot", symbol, logger)
}

func (s *StreamTickerBranch) Close() {
	(*s.cancel)()
	s.bid.mux.Lock()
	s.bid.price = NullPrice
	s.bid.mux.Unlock()
	s.ask.mux.Lock()
	s.ask.price = NullPrice
	s.ask.mux.Unlock()
}

func (s *StreamTickerBranch) GetBid() (price, qty string, ts time.Time, ok bool) {
	s.bid.mux.RLock()
	defer s.bid.mux.RUnlock()
	price = s.bid.price
	qty = s.bid.qty
	ts = s.bid.timestamp
	if price == NullPrice || price == "" {
		return price, qty, ts, false
	}
	return price, qty, ts, true
}

func (s *StreamTickerBranch) GetAsk() (price, qty string, ts time.Time, ok bool) {
	s.ask.mux.RLock()
	defer s.ask.mux.RUnlock()
	price = s.ask.price
	qty = s.ask.qty
	ts = s.bid.timestamp
	if price == NullPrice || price == "" {
		return price, qty, ts, false
	}
	return price, qty, ts, true
}

// internal

func localStreamTicker(product, symbol string, logger *log.Logger) *StreamTickerBranch {
	var s StreamTickerBranch
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = &cancel
	ticker := make(chan map[string]interface{}, 50)
	errCh := make(chan error, 5)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := s.socketTicker(ctx, product, symbol, logger, &ticker, &errCh); err == nil {
					return
				} else {
					logger.Warningf("Reconnect %s %s ticker stream with err: %s\n", symbol, product, err.Error())
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := s.maintainStreamTicker(ctx, product, symbol, &ticker, &errCh); err == nil {
					return
				} else {
					logger.Warningf("Refreshing %s %s ticker stream with err: %s\n", symbol, product, err.Error())
				}
			}
		}
	}()
	return &s
}

func (s *StreamTickerBranch) updateBidData(price, qty string, ts time.Time) {
	s.bid.mux.Lock()
	defer s.bid.mux.Unlock()
	s.bid.price = price
	s.bid.qty = qty
	s.bid.timestamp = ts
}

func (s *StreamTickerBranch) updateAskData(price, qty string, ts time.Time) {
	s.ask.mux.Lock()
	defer s.ask.mux.Unlock()
	s.ask.price = price
	s.ask.qty = qty
	s.ask.timestamp = ts
}

func (s *StreamTickerBranch) maintainStreamTicker(
	ctx context.Context,
	product, symbol string,
	ticker *chan map[string]interface{},
	errCh *chan error,
) error {
	lastUpdate := time.Now()
	for {
		select {
		case <-ctx.Done():
			return nil
		case message := <-(*ticker):
			var bidPrice, askPrice, bidQty, askQty string
			if bid, ok := message["b"].(string); ok {
				bidPrice = bid
			} else {
				bidPrice = NullPrice
			}
			if ask, ok := message["a"].(string); ok {
				askPrice = ask
			} else {
				askPrice = NullPrice
			}
			if bidqty, ok := message["B"].(string); ok {
				bidQty = bidqty
			}
			if askqty, ok := message["A"].(string); ok {
				askQty = askqty
			}
			var ts time.Time
			if eventTime, ok := message["E"].(float64); ok {
				ts = time.UnixMilli(int64(eventTime))
			}
			s.updateBidData(bidPrice, bidQty, ts)
			s.updateAskData(askPrice, askQty, ts)
			lastUpdate = time.Now()
		default:
			if time.Now().After(lastUpdate.Add(time.Second * 10)) {
				// 10 sec without updating
				err := errors.New("reconnect because of time out")
				*errCh <- err
				return err
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (s *StreamTickerBranch) socketTicker(
	ctx context.Context,
	product, symbol string,
	logger *log.Logger,
	mainCh *chan map[string]interface{},
	errCh *chan error,
) error {
	var duration time.Duration = 30
	s.socket.Logger = logger
	s.socket.OnErr = false
	var buffer bytes.Buffer
	switch product {
	case "perp":
		buffer.WriteString("wss://fstream3.binance.com/ws/")
		buffer.WriteString(strings.ToLower(symbol))
		buffer.WriteString("@bookTicker")
	default:
		buffer.WriteString("wss://stream.binance.com:9443/ws/")
		buffer.WriteString(strings.ToLower(symbol))
		buffer.WriteString("@ticker")
	}
	url := buffer.String()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	log.Println("Connected:", url)
	s.socket.Conn = conn
	defer conn.Close()
	if err := s.socket.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-*errCh:
			return err
		default:
			if s.socket.Conn == nil {
				s.outBinanceErr()
				message := "Binance reconnect..."
				log.Printf(message)
				return errors.New(message)
			}
			_, buf, err := conn.ReadMessage()
			if err != nil {
				s.outBinanceErr()
				message := "Binance reconnect..."
				log.Println(message)
				return errors.New(message)
			}
			res, err1 := decodingMap(buf, logger)
			if err1 != nil {
				s.outBinanceErr()
				message := "Binance reconnect..."
				log.Printf(message, err1)
				return err1
			}
			err2 := s.binanceHandleTickerHub(product, &res, mainCh)
			if err2 != nil {
				s.outBinanceErr()
				message := "Binance reconnect..."
				log.Printf(message, err2)
				return err2
			}
			if err := s.socket.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}

		}
	}
}

func (s *StreamTickerBranch) outBinanceErr() map[string]interface{} {
	s.socket.OnErr = true
	m := make(map[string]interface{})
	return m
}

func (w *StreamTickerBranch) binanceHandleTickerHub(product string, res *map[string]interface{}, mainCh *chan map[string]interface{}) error {
	if product == "perp" {
		err := w.handleBinancePerpTicker(res, mainCh)
		return err
	}
	err := w.handleBinanceSpotTicker(res, mainCh)
	return err
}

func (w *StreamTickerBranch) handleBinanceSpotTicker(res *map[string]interface{}, mainCh *chan map[string]interface{}) error {
	switch {
	case (*res)["e"] == "24hrTicker":
		Timestamp := formatingTimeStamp((*res)["E"].(float64))
		NowTime := time.Now()
		if NowTime.After(Timestamp.Add(time.Second*2)) == true {
			w.outBinanceErr()
			err := errors.New("websocket data delay more than 2 sec")
			return err
		}
		if !w.socket.OnErr {
			*mainCh <- *res
		}
		return nil
	}
	return errors.New("unsupport channel error")
}

func (w *StreamTickerBranch) handleBinancePerpTicker(res *map[string]interface{}, mainCh *chan map[string]interface{}) error {
	Timestamp := formatingTimeStamp((*res)["E"].(float64))
	NowTime := time.Now()
	if NowTime.After(Timestamp.Add(time.Second*2)) == true {
		w.outBinanceErr()
		err := errors.New("websocket data delay more than 2 sec")
		return err
	}
	if !w.socket.OnErr {
		*mainCh <- *res
	}
	return nil
}
