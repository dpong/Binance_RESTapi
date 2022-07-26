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

type OrderBookBranch struct {
	bids          bookBranch
	asks          bookBranch
	lastUpdatedId lastUpdateIdbranch
	snapShoted    bool
	cancel        *context.CancelFunc
	LookBack      time.Duration
	fromLevel     int
	toLevel       int
	reCh          chan error
	lastRefresh   lastRefreshBranch
}

func SpotLocalOrderBook(symbol string, logger *log.Logger) *OrderBookBranch {
	return localOrderBook("spot", symbol, logger)
}

func PerpLocalOrderBook(symbol string, logger *log.Logger) *OrderBookBranch {
	return localOrderBook("perp", symbol, logger)
}

func (o *OrderBookBranch) RefreshLocalOrderBook(err error) error {
	if o.ifCanRefresh() {
		if len(o.reCh) == cap(o.reCh) {
			return errors.New("refresh channel is full, please check it up")
		}
		o.reCh <- err
	}
	return nil
}

func (o *OrderBookBranch) Close() {
	(*o.cancel)()
	o.snapShoted = false
	o.bids.mux.Lock()
	o.bids.Book = [][]string{}
	o.bids.mux.Unlock()
	o.asks.mux.Lock()
	o.asks.Book = [][]string{}
	o.asks.mux.Unlock()
}

// return bids, ready or not
func (o *OrderBookBranch) GetBids() ([][]string, bool) {
	o.bids.mux.RLock()
	defer o.bids.mux.RUnlock()
	if !o.snapShoted {
		return [][]string{}, false
	}
	if len(o.bids.Book) == 0 {
		if o.ifCanRefresh() {
			o.reCh <- errors.New("re cause len bid is zero")
		}
		return [][]string{}, false
	}
	book := o.bids.Book
	return book, true
}

// return asks, ready or not
func (o *OrderBookBranch) GetAsks() ([][]string, bool) {
	o.asks.mux.RLock()
	defer o.asks.mux.RUnlock()
	if !o.snapShoted {
		return [][]string{}, false
	}
	if len(o.asks.Book) == 0 {
		if o.ifCanRefresh() {
			o.reCh <- errors.New("re cause len ask is zero")
		}
		return [][]string{}, false
	}
	book := o.asks.Book
	return book, true
}

func (o *OrderBookBranch) SetLookBackSec(input int) {
	o.LookBack = time.Duration(input) * time.Second
}

// internal

type lastUpdateIdbranch struct {
	mux sync.RWMutex
	ID  decimal.Decimal
}

type lastRefreshBranch struct {
	mux  sync.RWMutex
	time time.Time
}

type bookBranch struct {
	mux  sync.RWMutex
	Book [][]string
}

type wS struct {
	Channel       string
	Logger        *log.Logger
	Conn          *websocket.Conn
	OnErr         bool
	LastUpdatedId decimal.Decimal
}

func (o *OrderBookBranch) ifCanRefresh() bool {
	o.lastRefresh.mux.Lock()
	defer o.lastRefresh.mux.Unlock()
	now := time.Now()
	if now.After(o.lastRefresh.time.Add(time.Second * 3)) {
		o.lastRefresh.time = now
		return true
	}
	return false
}

// logurs as log system
func (o *OrderBookBranch) getOrderBookSnapShot(product, symbol string) error {
	client := New("", "", "")
	switch product {
	case "spot":
		res, err := client.SpotDepth(symbol, 5000)
		if err != nil {
			return err
		}
		o.bids.mux.Lock()
		o.bids.Book = res.Bids
		o.bids.mux.Unlock()
		o.asks.mux.Lock()
		o.asks.Book = res.Asks
		o.asks.mux.Unlock()
		o.updateLastUpdateId(decimal.NewFromInt(int64(res.LastUpdateID)))
	case "perp":
		res, err := client.SwapDepth(symbol, 1000)
		if err != nil {
			return err
		}
		o.bids.mux.Lock()
		o.bids.Book = res.Bids
		o.bids.mux.Unlock()
		o.asks.mux.Lock()
		o.asks.Book = res.Asks
		o.asks.mux.Unlock()
		o.updateLastUpdateId(decimal.NewFromInt(int64(res.LastUpdateID)))
	}
	o.snapShoted = true
	return nil
}

func (o *OrderBookBranch) updateNewComing(message *map[string]interface{}) {
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
			o.dealWithBidPriceLevel(price, qty)
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
			o.dealWithAskPriceLevel(price, qty)
		}
	}()
	wg.Wait()
}

func (o *OrderBookBranch) dealWithBidPriceLevel(price, qty decimal.Decimal) {
	o.bids.mux.Lock()
	defer o.bids.mux.Unlock()
	l := len(o.bids.Book)
	for level, item := range o.bids.Book {
		bookPrice, _ := decimal.NewFromString(item[0])
		switch {
		case price.GreaterThan(bookPrice):
			// insert level
			if qty.IsZero() {
				// ignore
				return
			}
			o.bids.Book = append(o.bids.Book, []string{})
			copy(o.bids.Book[level+1:], o.bids.Book[level:])
			o.bids.Book[level] = []string{price.String(), qty.String()}
			return
		case price.LessThan(bookPrice):
			if level == l-1 {
				// insert last level
				if qty.IsZero() {
					// ignore
					return
				}
				o.bids.Book = append(o.bids.Book, []string{price.String(), qty.String()})
				return
			}
			continue
		case price.Equal(bookPrice):
			if qty.IsZero() {
				// delete level
				switch {
				case level == l-1:
					o.bids.Book = o.bids.Book[:l-1]
				default:
					o.bids.Book = append(o.bids.Book[:level], o.bids.Book[level+1:]...)
				}
				return
			}
			o.bids.Book[level][1] = qty.String()
			return
		}
	}
}

func (o *OrderBookBranch) dealWithAskPriceLevel(price, qty decimal.Decimal) {
	o.asks.mux.Lock()
	defer o.asks.mux.Unlock()
	l := len(o.asks.Book)
	for level, item := range o.asks.Book {
		bookPrice, _ := decimal.NewFromString(item[0])
		switch {
		case price.LessThan(bookPrice):
			// insert level
			if qty.IsZero() {
				// ignore
				return
			}
			o.asks.Book = append(o.asks.Book, []string{})
			copy(o.asks.Book[level+1:], o.asks.Book[level:])
			o.asks.Book[level] = []string{price.String(), qty.String()}
			return
		case price.GreaterThan(bookPrice):
			if level == l-1 {
				// insert last level
				if qty.IsZero() {
					// ignore
					return
				}
				o.asks.Book = append(o.asks.Book, []string{price.String(), qty.String()})
				return
			}
			continue
		case price.Equal(bookPrice):
			if qty.IsZero() {
				// delete level
				switch {
				case level == l-1:
					o.asks.Book = o.asks.Book[:l-1]
				default:
					o.asks.Book = append(o.asks.Book[:level], o.asks.Book[level+1:]...)
				}
				return
			}
			o.asks.Book[level][1] = qty.String()
			return
		}
	}
}

func (o *OrderBookBranch) updateLastUpdateId(id decimal.Decimal) {
	o.lastUpdatedId.mux.Lock()
	defer o.lastUpdatedId.mux.Unlock()
	o.lastUpdatedId.ID = id
}

func (o *OrderBookBranch) readLastUpdateId() decimal.Decimal {
	o.lastUpdatedId.mux.RLock()
	defer o.lastUpdatedId.mux.RUnlock()
	return o.lastUpdatedId.ID
}

func reStartMainSeesionErrHub(err string) bool {
	switch {
	case strings.Contains(err, "reconnect because of time out"):
		return false
	case strings.Contains(err, "reconnect because of reCh send"):
		return false
	case strings.Contains(err, "reconnect because of snapshot fail"):
		return false
	}
	return true
}

func localOrderBook(product, symbol string, logger *log.Logger) *OrderBookBranch {
	var o OrderBookBranch
	o.SetLookBackSec(5)
	ctx, cancel := context.WithCancel(context.Background())
	o.cancel = &cancel
	bookticker := make(chan map[string]interface{}, 50)
	errCh := make(chan error, 1)
	o.reCh = make(chan error, 5)
	symbol = strings.ToUpper(symbol)
	// stream orderbook
	orderBookErr := make(chan error, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := binanceSocket(ctx, product, symbol, "@depth@100ms", logger, &bookticker, &orderBookErr); err == nil {
					return
				} else {
					if reStartMainSeesionErrHub(err.Error()) {
						errCh <- errors.New("Reconnect websocket")
					}
					logger.Warningf("Reconnect %s %s orderbook stream.\n", symbol, product)
					//time.Sleep(time.Second)
				}
			}
		}
	}()
	// stream trade
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := o.maintainOrderBook(ctx, product, symbol, &bookticker, &errCh, &orderBookErr)
				if err == nil {
					return
				}
				logger.Warningf("Refreshing %s %s local orderbook cause: %s\n", symbol, product, err.Error())
				//time.Sleep(time.Second)
			}
		}
	}()
	return &o
}

func (o *OrderBookBranch) maintainOrderBook(
	ctx context.Context,
	product, symbol string,
	bookticker *chan map[string]interface{},
	errCh *chan error,
	orderBookErr *chan error,
) error {
	var storage []map[string]interface{}
	var linked bool = false
	o.snapShoted = false
	o.updateLastUpdateId(decimal.Zero)
	lastUpdate := time.Now()
	snapshotErr := make(chan error, 1)
	go func() {
		// avoid latancy issue
		time.Sleep(time.Second * 3)
		if err := o.getOrderBookSnapShot(product, symbol); err != nil {
			snapshotErr <- err
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-(*errCh):
			return err
		case err := <-snapshotErr:
			errSend := errors.New("reconnect because of snapshot fail")
			(*orderBookErr) <- errSend
			return err
		case err := <-o.reCh:
			errSend := errors.New("reconnect because of reCh send")
			(*orderBookErr) <- errSend
			return err
		case message := <-(*bookticker):
			event, ok := message["e"].(string)
			if !ok {
				continue
			}
			switch event {
			case "depthUpdate":
				if !o.snapShoted {
					storage = append(storage, message)
					continue
				}
				if len(storage) != 0 {
					for _, data := range storage {
						switch product {
						case "spot":
							if err := o.spotUpdateJudge(&data, &linked); err != nil {
								return err
							}
						case "perp":
							if err := o.swapUpdateJudge(&data, &linked); err != nil {
								return err
							}
						}
					}
					// clear storage
					storage = make([]map[string]interface{}, 0)
				}
				// handle incoming data
				switch product {
				case "spot":
					if err := o.spotUpdateJudge(&message, &linked); err != nil {
						return err
					}
				case "perp":
					if err := o.swapUpdateJudge(&message, &linked); err != nil {
						return err
					}
				}
				// update last update
				lastUpdate = time.Now()
			}
		default:
			if time.Now().After(lastUpdate.Add(time.Second * 10)) {
				// 10 sec without updating
				err := errors.New("reconnect because of time out")
				(*orderBookErr) <- err
				return err
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (o *OrderBookBranch) spotUpdateJudge(message *map[string]interface{}, linked *bool) error {
	headID := decimal.NewFromFloat((*message)["U"].(float64))
	tailID := decimal.NewFromFloat((*message)["u"].(float64))
	baseID := o.readLastUpdateId()
	snapID := baseID.Add(decimal.NewFromInt(1))
	if !(*linked) {
		switch {
		case headID.LessThanOrEqual(snapID) && tailID.GreaterThanOrEqual(snapID):
			//U <= lastUpdateId+1 AND u >= lastUpdateId+1.
			(*linked) = true
			o.updateNewComing(message)
			o.updateLastUpdateId(tailID)
		case tailID.LessThan(snapID):
			// drop pre data
		default:
			// latancy issue, reconnect
			return errors.New("refresh.")
		}
	} else {
		if headID.Equal(snapID) {
			o.updateNewComing(message)
			o.updateLastUpdateId(tailID)
		} else {
			return errors.New("refresh.")
		}
	}
	return nil
}

func (o *OrderBookBranch) swapUpdateJudge(message *map[string]interface{}, linked *bool) error {
	headID := decimal.NewFromFloat((*message)["U"].(float64))
	tailID := decimal.NewFromFloat((*message)["u"].(float64))
	puID := decimal.NewFromFloat((*message)["pu"].(float64))
	snapID := o.readLastUpdateId()
	if !(*linked) {
		// drop u is < lastUpdateId
		if tailID.LessThan(snapID) {
			return nil
		}
		// U <= lastUpdateId AND u >= lastUpdateId
		if headID.LessThanOrEqual(snapID) && tailID.GreaterThanOrEqual(snapID) {
			(*linked) = true
			o.updateNewComing(message)
			o.updateLastUpdateId(tailID)
		}
	} else {
		// new event's pu should be equal to the previous event's u
		if puID.Equal(snapID) {
			o.updateNewComing(message)
			o.updateLastUpdateId(tailID)
		} else {
			return errors.New("refresh.")
		}
	}
	return nil
}

func decodingMap(message []byte, logger *log.Logger) (res map[string]interface{}, err error) {
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

func binanceSocket(ctx context.Context, product, symbol, channel string, logger *log.Logger, mainCh *chan map[string]interface{}, reCh *chan error) error {
	var w wS
	var duration time.Duration = 300
	w.Channel = channel
	w.Logger = logger
	var buffer bytes.Buffer
	switch product {
	case "spot":
		buffer.WriteString("wss://stream.binance.com:9443/ws/")
	case "perp":
		buffer.WriteString("wss://fstream.binance.com/ws/")
	}
	buffer.WriteString(strings.ToLower(symbol))
	buffer.WriteString(w.Channel)
	url := buffer.String()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	logger.Infof("Binance %s %s %s socket connected.\n", symbol, product, channel)
	w.Conn = conn
	defer conn.Close()
	if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
		return err
	}
	w.Conn.SetPingHandler(nil)
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-(*reCh):
			return err
		default:
			if w.Conn == nil {
				d := w.outBinanceErr()
				*mainCh <- d
				message := "Binance reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			_, buf, err := conn.ReadMessage()
			if err != nil {
				d := w.outBinanceErr()
				*mainCh <- d
				message := "Binance reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			res, err1 := decodingMap(buf, logger)
			if err1 != nil {
				d := w.outBinanceErr()
				*mainCh <- d
				message := "Binance reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			err2 := w.handleBinanceSocketData(res, mainCh)
			if err2 != nil {
				d := w.outBinanceErr()
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

func (w *wS) handleBinanceSocketData(res map[string]interface{}, mainCh *chan map[string]interface{}) error {
	event, ok := res["e"].(string)
	if !ok {
		return nil
	}
	switch event {
	case "depthUpdate":
		if st, ok := res["E"].(float64); !ok {
			m := w.outBinanceErr()
			*mainCh <- m
			return errors.New("got nil when updating event time")
		} else {
			stamp := formatingTimeStamp(st)
			if time.Now().After(stamp.Add(time.Second * 5)) {
				m := w.outBinanceErr()
				*mainCh <- m
				return errors.New("websocket data delay more than 5 sec")
			}
		}
		firstId := res["U"].(float64)
		lastId := res["u"].(float64)
		headID := decimal.NewFromFloat(firstId)
		tailID := decimal.NewFromFloat(lastId)
		if headID.LessThan(w.LastUpdatedId) {
			m := w.outBinanceErr()
			*mainCh <- m
			return errors.New("got error when updating lastUpdateId")
		}
		w.LastUpdatedId = tailID
		*mainCh <- res
	case "trade":
		*mainCh <- res
	case "aggTrade":
		*mainCh <- res
	}
	return nil
}

func (w *wS) outBinanceErr() map[string]interface{} {
	m := make(map[string]interface{})
	return m
}

func formatingTimeStamp(timeFloat float64) time.Time {
	t := time.Unix(int64(timeFloat/1000), 0)
	return t
}
