package bnnapi

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type OrderBookBranch struct {
	Bids          BookBranch
	Asks          BookBranch
	LastUpdatedId decimal.Decimal
	SnapShoted    bool
	Cancel        *context.CancelFunc
}

type BookBranch struct {
	mux  sync.RWMutex
	Book [][]string
}

type WS struct {
	Channel       string
	OnErr         bool
	Logger        *log.Logger
	Conn          *websocket.Conn
	LastUpdatedId decimal.Decimal
}

// logurs as log system
func (o *OrderBookBranch) GetOrderBookSnapShot(symbol string) error {
	client := New("", "", "")
	res, err := client.SpotDepth(symbol, 5000)
	if err != nil {
		return err
	}
	o.Bids.mux.Lock()
	o.Bids.Book = res.Bids
	o.Bids.mux.Unlock()
	o.Asks.mux.Lock()
	o.Asks.Book = res.Asks
	o.Asks.mux.Unlock()
	o.LastUpdatedId = decimal.NewFromInt(int64(res.LastUpdateID))
	o.SnapShoted = true
	return nil
}

func (o *OrderBookBranch) UpdateNewComing(message *map[string]interface{}) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		// bid
		bids, ok := (*message)["b"].([]interface{})
		if !ok {
			return
		}
		for _, bid := range bids {
			price, _ := decimal.NewFromString(bid.([]interface{})[0].(string))
			qty, _ := decimal.NewFromString(bid.([]interface{})[1].(string))
			o.DealWithBidPriceLevel(price, qty)
		}
	}()
	go func() {
		defer wg.Done()
		// ask
		asks, ok := (*message)["a"].([]interface{})
		if !ok {
			return
		}
		for _, ask := range asks {
			price, _ := decimal.NewFromString(ask.([]interface{})[0].(string))
			qty, _ := decimal.NewFromString(ask.([]interface{})[1].(string))
			o.DealWithAskPriceLevel(price, qty)
		}
	}()
	wg.Wait()
}

func (o *OrderBookBranch) DealWithBidPriceLevel(price, qty decimal.Decimal) {
	o.Bids.mux.Lock()
	defer o.Bids.mux.Unlock()
	l := len(o.Bids.Book)
	for level, item := range o.Bids.Book {
		bookPrice, _ := decimal.NewFromString(item[0])
		switch {
		case price.GreaterThan(bookPrice):
			// insert level
			if qty.IsZero() {
				// ignore
				return
			}
			o.Bids.Book = append(o.Bids.Book, []string{})
			copy(o.Bids.Book[level+1:], o.Bids.Book[level:])
			o.Bids.Book[level] = []string{price.String(), qty.String()}
			return
		case price.LessThan(bookPrice):
			if level == l-1 {
				// insert last level
				if qty.IsZero() {
					// ignore
					return
				}
				o.Bids.Book = append(o.Bids.Book, []string{price.String(), qty.String()})
				return
			}
			continue
		case price.Equal(bookPrice):
			if qty.IsZero() {
				// delete level
				o.Bids.Book = append(o.Bids.Book[:level], o.Bids.Book[level+1:]...)
				return
			}
			o.Bids.Book[level][1] = qty.String()
			return
		}
	}
}

func (o *OrderBookBranch) DealWithAskPriceLevel(price, qty decimal.Decimal) {
	o.Asks.mux.Lock()
	defer o.Asks.mux.Unlock()
	l := len(o.Asks.Book)
	for level, item := range o.Asks.Book {
		bookPrice, _ := decimal.NewFromString(item[0])
		switch {
		case price.LessThan(bookPrice):
			// insert level
			if qty.IsZero() {
				// ignore
				return
			}
			o.Asks.Book = append(o.Asks.Book, []string{})
			copy(o.Asks.Book[level+1:], o.Asks.Book[level:])
			o.Asks.Book[level] = []string{price.String(), qty.String()}
			return
		case price.GreaterThan(bookPrice):
			if level == l-1 {
				// insert last level
				if qty.IsZero() {
					// ignore
					return
				}
				o.Asks.Book = append(o.Asks.Book, []string{price.String(), qty.String()})
				return
			}
			continue
		case price.Equal(bookPrice):
			if qty.IsZero() {
				// delete level
				o.Asks.Book = append(o.Asks.Book[:level], o.Asks.Book[level+1:]...)
				return
			}
			o.Asks.Book[level][1] = qty.String()
			return
		}
	}
}

func (o *OrderBookBranch) Close() {
	(*o.Cancel)()
	o.SnapShoted = true
	o.Bids.mux.Lock()
	o.Bids.Book = [][]string{}
	o.Bids.mux.Unlock()
	o.Asks.mux.Lock()
	o.Asks.Book = [][]string{}
	o.Asks.mux.Unlock()
}

// return bids, ready or not
func (o *OrderBookBranch) GetBids() ([][]string, bool) {
	o.Bids.mux.RLock()
	defer o.Bids.mux.RUnlock()
	if len(o.Bids.Book) == 0 || !o.SnapShoted {
		return [][]string{}, false
	}
	book := o.Bids.Book
	return book, true
}

// return asks, ready or not
func (o *OrderBookBranch) GetAsks() ([][]string, bool) {
	o.Asks.mux.RLock()
	defer o.Asks.mux.RUnlock()
	if len(o.Asks.Book) == 0 || !o.SnapShoted {
		return [][]string{}, false
	}
	book := o.Asks.Book
	return book, true
}

func BinanceLocalOrderBook(symbol string, logger *log.Logger) *OrderBookBranch {
	var o OrderBookBranch
	ctx, cancel := context.WithCancel(context.Background())
	o.Cancel = &cancel
	bookticker := make(chan map[string]interface{}, 50)
	errCh := make(chan error, 5)
	go func(logger *log.Logger, bookticker *chan map[string]interface{}) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := BinanceSocket(ctx, "spot", "BTCUSDT", "@depth@100ms", logger, bookticker); err == nil {
					return
				}
				errCh <- errors.New("Reconnect websocket")
				time.Sleep(time.Second)
			}
		}
	}(logger, &bookticker)
	go func() {
		for {
			select {
			case <-ctx.Done():
			default:
				o.MaintainOrderBook(ctx, symbol, &bookticker, &errCh)
				time.Sleep(time.Second)
			}
		}
	}()
	return &o
}

func (o *OrderBookBranch) MaintainOrderBook(ctx context.Context, symbol string, bookticker *chan map[string]interface{}, errCh *chan error) {
	var storage []map[string]interface{}
	go func() {
		time.Sleep(time.Second * 3)
		o.GetOrderBookSnapShot(symbol)
		o.SnapShoted = true
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-(*errCh):
			return
		default:
			message := <-(*bookticker)
			if len(message) != 0 {
				storage = append(storage, message)
				if !o.SnapShoted {
					continue
				}
				if len(storage) > 1 {
					for _, data := range storage {
						headID := decimal.NewFromFloat(data["U"].(float64))
						tailID := decimal.NewFromFloat(data["u"].(float64))
						snapID := o.LastUpdatedId.Add(decimal.NewFromInt(1))
						//U <= lastUpdateId+1 AND u >= lastUpdateId+1.
						if headID.LessThanOrEqual(snapID) && tailID.GreaterThanOrEqual(snapID) {
							// handle incoming data
							o.UpdateNewComing(&message)
							o.LastUpdatedId = tailID
						}
						storage = storage[1:]
					}
				}

				// handle incoming data
				headID := decimal.NewFromFloat(message["U"].(float64))
				tailID := decimal.NewFromFloat(message["u"].(float64))
				snapID := o.LastUpdatedId.Add(decimal.NewFromInt(1))
				if headID.LessThanOrEqual(snapID) && tailID.GreaterThanOrEqual(snapID) {
					o.UpdateNewComing(&message)
					o.LastUpdatedId = tailID
				}
			}
		}
	}
}

func DecodingMap(message []byte, logger *log.Logger) (res map[string]interface{}, err error) {
	if message == nil {
		err = errors.New("the incoming message is nil")
		return nil, err
	}
	err = json.Unmarshal(message, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func BinanceSocket(ctx context.Context, product, symbol, channel string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w WS
	var duration time.Duration = 30
	w.Channel = channel
	w.Logger = logger
	w.OnErr = false
	var buffer bytes.Buffer
	switch product {
	case "spot":
		buffer.WriteString("wss://stream.binance.com:9443/ws/")
	case "swap":
		buffer.WriteString("wss://fstream3.binance.com/ws/")
	}
	buffer.WriteString(strings.ToLower(symbol))
	buffer.WriteString(w.Channel)
	url := buffer.String()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	logger.Infoln("Connected:", url)
	w.Conn = conn
	defer conn.Close()
	if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if w.Conn == nil {
				d := w.OutBinanceErr()
				*mainCh <- d
				message := "Binance reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			_, buf, err := conn.ReadMessage()
			if err != nil {
				d := w.OutBinanceErr()
				*mainCh <- d
				message := "Binance reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			res, err1 := DecodingMap(buf, logger)
			if err1 != nil {
				d := w.OutBinanceErr()
				*mainCh <- d
				message := "Binance reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			err2 := w.HandleBinanceSocketData(res, mainCh)
			if err2 != nil {
				d := w.OutBinanceErr()
				*mainCh <- d
				message := "Binance reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
		}
	}
}

func (w *WS) HandleBinanceSocketData(res map[string]interface{}, mainCh *chan map[string]interface{}) error {
	firstId := res["U"].(float64)
	lastId := res["u"].(float64)
	headID := decimal.NewFromFloat(firstId)
	tailID := decimal.NewFromFloat(lastId)
	if headID.LessThan(w.LastUpdatedId) {
		m := w.OutBinanceErr()
		*mainCh <- m
		return errors.New("got error when updating lastUpdateId")
	}
	w.LastUpdatedId = tailID
	*mainCh <- res
	return nil
}

func (w *WS) OutBinanceErr() map[string]interface{} {
	w.OnErr = true
	m := make(map[string]interface{})
	return m
}

func FormatingTimeStamp(timeFloat float64) time.Time {
	sec, dec := math.Modf(timeFloat)
	return time.Unix(int64(sec), int64(dec*(1e9)))
}
