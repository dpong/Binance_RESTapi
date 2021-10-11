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
	Bids          BookBranch
	Asks          BookBranch
	LastUpdatedId decimal.Decimal
	SnapShoted    bool
	Cancel        *context.CancelFunc
	BuyTrade      TradeImpact
	SellTrade     TradeImpact
	LookBack      time.Duration
	fromLevel     int
	toLevel       int
}

type TradeImpact struct {
	mux      sync.RWMutex
	Stamp    []time.Time
	Qty      []decimal.Decimal
	Notional []decimal.Decimal
}

type BookBranch struct {
	mux   sync.RWMutex
	Book  [][]string
	Micro []BookMicro
}

type BookMicro struct {
	OrderNum int
	Trend    string
}

type WS struct {
	Channel       string
	OnErr         bool
	Logger        *log.Logger
	Conn          *websocket.Conn
	LastUpdatedId decimal.Decimal
}

// logurs as log system
func (o *OrderBookBranch) GetOrderBookSnapShot(product, symbol string) error {
	client := New("", "", "")
	switch product {
	case "spot":
		res, err := client.SpotDepth(symbol, 5000)
		if err != nil {
			return err
		}
		o.Bids.mux.Lock()
		o.Bids.Book = res.Bids
		for i := 0; i < len(res.Bids); i++ {
			// micro part
			micro := BookMicro{
				OrderNum: 1, // initial order num is 1
			}
			o.Bids.Micro = append(o.Bids.Micro, micro)
		}
		o.Bids.mux.Unlock()
		o.Asks.mux.Lock()
		o.Asks.Book = res.Asks
		for i := 0; i < len(res.Asks); i++ {
			// micro part
			micro := BookMicro{
				OrderNum: 1, // initial order num is 1
			}
			o.Asks.Micro = append(o.Asks.Micro, micro)
		}
		o.Asks.mux.Unlock()
		o.LastUpdatedId = decimal.NewFromInt(int64(res.LastUpdateID))
	case "swap":
		res, err := client.SwapDepth(symbol, 1000)
		if err != nil {
			return err
		}
		o.Bids.mux.Lock()
		o.Bids.Book = res.Bids
		for i := 0; i < len(res.Bids); i++ {
			// micro part
			micro := BookMicro{
				OrderNum: 1, // initial order num is 1
			}
			o.Bids.Micro = append(o.Bids.Micro, micro)
		}
		o.Bids.mux.Unlock()
		o.Asks.mux.Lock()
		o.Asks.Book = res.Asks
		for i := 0; i < len(res.Asks); i++ {
			// micro part
			micro := BookMicro{
				OrderNum: 1, // initial order num is 1
			}
			o.Asks.Micro = append(o.Asks.Micro, micro)
		}
		o.Asks.mux.Unlock()
		o.LastUpdatedId = decimal.NewFromInt(int64(res.LastUpdateID))
	}
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
			// micro part
			o.Bids.Micro = append(o.Bids.Micro, BookMicro{})
			copy(o.Bids.Micro[level+1:], o.Bids.Micro[level:])
			o.Bids.Book[level] = []string{price.String(), qty.String()}
			o.Bids.Micro[level].OrderNum = 1
			return
		case price.LessThan(bookPrice):
			if level == l-1 {
				// insert last level
				if qty.IsZero() {
					// ignore
					return
				}
				o.Bids.Book = append(o.Bids.Book, []string{price.String(), qty.String()})
				data := BookMicro{
					OrderNum: 1,
				}
				o.Bids.Micro = append(o.Bids.Micro, data)
				return
			}
			continue
		case price.Equal(bookPrice):
			if qty.IsZero() {
				// delete level
				switch {
				case level == l-1:
					o.Bids.Book = o.Bids.Book[:l-1]
					o.Bids.Micro = o.Bids.Micro[:l-1]
				default:
					o.Bids.Book = append(o.Bids.Book[:level], o.Bids.Book[level+1:]...)
					o.Bids.Micro = append(o.Bids.Micro[:level], o.Bids.Micro[level+1:]...)
				}
				return
			}
			oldQty, _ := decimal.NewFromString(o.Bids.Book[level][1])
			switch {
			case oldQty.GreaterThan(qty):
				// add order
				o.Bids.Micro[level].OrderNum++
				o.Bids.Micro[level].Trend = "add"
			case oldQty.LessThan(qty):
				// cut order
				o.Bids.Micro[level].OrderNum--
				o.Bids.Micro[level].Trend = "cut"
				if o.Bids.Micro[level].OrderNum < 1 {
					o.Bids.Micro[level].OrderNum = 1
				}
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
			// micro part
			o.Asks.Micro = append(o.Asks.Micro, BookMicro{})
			copy(o.Asks.Micro[level+1:], o.Asks.Micro[level:])
			o.Asks.Book[level] = []string{price.String(), qty.String()}
			o.Asks.Micro[level].OrderNum = 1
			return
		case price.GreaterThan(bookPrice):
			if level == l-1 {
				// insert last level
				if qty.IsZero() {
					// ignore
					return
				}
				o.Asks.Book = append(o.Asks.Book, []string{price.String(), qty.String()})
				data := BookMicro{
					OrderNum: 1,
				}
				o.Asks.Micro = append(o.Asks.Micro, data)
				return
			}
			continue
		case price.Equal(bookPrice):
			if qty.IsZero() {
				// delete level
				switch {
				case level == l-1:
					o.Asks.Book = o.Asks.Book[:l-1]
					o.Asks.Micro = o.Asks.Micro[:l-1]
				default:
					o.Asks.Book = append(o.Asks.Book[:level], o.Asks.Book[level+1:]...)
					o.Asks.Micro = append(o.Asks.Micro[:level], o.Asks.Micro[level+1:]...)
				}
				return
			}
			oldQty, _ := decimal.NewFromString(o.Asks.Book[level][1])
			switch {
			case oldQty.GreaterThan(qty):
				// add order
				o.Asks.Micro[level].OrderNum++
				o.Asks.Micro[level].Trend = "add"
			case oldQty.LessThan(qty):
				// cut order
				o.Asks.Micro[level].OrderNum--
				o.Asks.Micro[level].Trend = "cut"
				if o.Asks.Micro[level].OrderNum < 1 {
					o.Asks.Micro[level].OrderNum = 1
				}
			}
			o.Asks.Book[level][1] = qty.String()
			return
		}
	}
}

func (o *OrderBookBranch) Close() {
	(*o.Cancel)()
	o.SnapShoted = false
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

func (o *OrderBookBranch) GetBidMicro(idx int) (*BookMicro, bool) {
	o.Bids.mux.RLock()
	defer o.Bids.mux.RUnlock()
	if len(o.Bids.Book) == 0 || !o.SnapShoted {
		return nil, false
	}
	micro := o.Bids.Micro[idx]
	return &micro, true
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

func (o *OrderBookBranch) GetAskMicro(idx int) (*BookMicro, bool) {
	o.Asks.mux.RLock()
	defer o.Asks.mux.RUnlock()
	if len(o.Asks.Book) == 0 || !o.SnapShoted {
		return nil, false
	}
	micro := o.Asks.Micro[idx]
	return &micro, true
}

func (o *OrderBookBranch) GetBuyImpactNotion() decimal.Decimal {
	o.BuyTrade.mux.RLock()
	defer o.BuyTrade.mux.RUnlock()
	var total decimal.Decimal
	now := time.Now()
	for i, st := range o.BuyTrade.Stamp {
		if now.After(st.Add(o.LookBack)) {
			continue
		}
		total = total.Add(o.BuyTrade.Notional[i])
	}
	return total
}

func (o *OrderBookBranch) GetSellImpactNotion() decimal.Decimal {
	o.SellTrade.mux.RLock()
	defer o.SellTrade.mux.RUnlock()
	var total decimal.Decimal
	now := time.Now()
	for i, st := range o.SellTrade.Stamp {
		if now.After(st.Add(o.LookBack)) {
			continue
		}
		total = total.Add(o.SellTrade.Notional[i])
	}
	return total
}

func (o *OrderBookBranch) CalBidCumNotional() (decimal.Decimal, bool) {
	if len(o.Bids.Book) == 0 {
		return decimal.NewFromFloat(0), false
	}
	if o.fromLevel > o.toLevel {
		return decimal.NewFromFloat(0), false
	}
	o.Bids.mux.RLock()
	defer o.Bids.mux.RUnlock()
	var total decimal.Decimal
	for level, item := range o.Bids.Book {
		if level >= o.fromLevel && level <= o.toLevel {
			price, _ := decimal.NewFromString(item[0])
			qty, _ := decimal.NewFromString(item[1])
			total = total.Add(qty.Mul(price))
		} else if level > o.toLevel {
			break
		}
	}
	return total, true
}

func (o *OrderBookBranch) CalAskCumNotional() (decimal.Decimal, bool) {
	if len(o.Asks.Book) == 0 {
		return decimal.NewFromFloat(0), false
	}
	if o.fromLevel > o.toLevel {
		return decimal.NewFromFloat(0), false
	}
	o.Asks.mux.RLock()
	defer o.Asks.mux.RUnlock()
	var total decimal.Decimal
	for level, item := range o.Asks.Book {
		if level >= o.fromLevel && level <= o.toLevel {
			price, _ := decimal.NewFromString(item[0])
			qty, _ := decimal.NewFromString(item[1])
			total = total.Add(qty.Mul(price))
		} else if level > o.toLevel {
			break
		}
	}
	return total, true
}

func (o *OrderBookBranch) IsBigImpactOnBid() bool {
	impact := o.GetSellImpactNotion()
	rest, ok := o.CalBidCumNotional()
	if !ok {
		return false
	}
	micro, ok := o.GetBidMicro(o.fromLevel)
	if !ok {
		return false
	}
	if impact.GreaterThanOrEqual(rest) && micro.Trend == "cut" {
		return true
	}
	return false
}

func (o *OrderBookBranch) IsBigImpactOnAsk() bool {
	impact := o.GetBuyImpactNotion()
	rest, ok := o.CalAskCumNotional()
	if !ok {
		return false
	}
	micro, ok := o.GetAskMicro(o.fromLevel)
	if !ok {
		return false
	}
	if impact.GreaterThanOrEqual(rest) && micro.Trend == "cut" {
		return true
	}
	return false
}

func (o *OrderBookBranch) SetLookBackSec(input int) {
	o.LookBack = time.Duration(input) * time.Second
}

// top of the book is 1, to the level you want to sum all the notions
func (o *OrderBookBranch) SetImpactCumRange(toLevel int) {
	o.fromLevel = 0
	o.toLevel = toLevel - 1
}

func LocalOrderBook(product, symbol string, logger *log.Logger) *OrderBookBranch {
	var o OrderBookBranch
	o.SetLookBackSec(5)
	o.SetImpactCumRange(20)
	ctx, cancel := context.WithCancel(context.Background())
	o.Cancel = &cancel
	bookticker := make(chan map[string]interface{}, 50)
	errCh := make(chan error, 5)
	symbol = strings.ToUpper(symbol)
	// stream orderbook
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := BinanceSocket(ctx, product, symbol, "@depth@100ms", logger, &bookticker); err == nil {
					return
				}
				errCh <- errors.New("Reconnect websocket")
				time.Sleep(time.Second)
			}
		}
	}()
	// stream trade
	var tradeChannel string
	switch product {
	case "spot":
		tradeChannel = "@trade"
	case "swap":
		tradeChannel = "@aggTrade"
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := BinanceSocket(ctx, product, symbol, tradeChannel, logger, &bookticker); err == nil {
					return
				}
				errCh <- errors.New("Reconnect websocket")
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
				o.MaintainOrderBook(ctx, product, symbol, &bookticker, &errCh)
				logger.Warningf("Refreshing %s %s local orderbook.\n", symbol, product)
				time.Sleep(time.Second)
			}
		}
	}()
	return &o
}

// default with look back 5 sec, impact range from 0 to 10 levels of the orderbook
func SpotLocalOrderBook(symbol string, logger *log.Logger) *OrderBookBranch {
	return LocalOrderBook("spot", symbol, logger)
}

// default with look back 5 sec, impact range from 0 to 10 levels of the orderbook
func SwapLocalOrderBook(symbol string, logger *log.Logger) *OrderBookBranch {
	return LocalOrderBook("swap", symbol, logger)
}

func (o *OrderBookBranch) MaintainOrderBook(
	ctx context.Context,
	product, symbol string,
	bookticker *chan map[string]interface{},
	errCh *chan error,
) {
	var storage []map[string]interface{}
	var linked bool = false
	o.SnapShoted = false
	o.LastUpdatedId = decimal.NewFromInt(0)
	go func() {
		time.Sleep(time.Second)
		o.GetOrderBookSnapShot(product, symbol)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-(*errCh):
			return
		default:
			message := <-(*bookticker)
			event, ok := message["e"].(string)
			if !ok {
				continue
			}
			switch event {
			case "depthUpdate":
				if !o.SnapShoted {
					storage = append(storage, message)
					continue
				}
				if len(storage) != 0 {
					for _, data := range storage {
						switch product {
						case "spot":
							if err := o.SpotUpdateJudge(&data, &linked); err != nil {
								*errCh <- err
							}
						case "swap":
							if err := o.SwapUpdateJudge(&data, &linked); err != nil {
								*errCh <- err
							}
						}
					}
					// clear storage
					storage = make([]map[string]interface{}, 0)
				}
				// handle incoming data
				switch product {
				case "spot":
					if err := o.SpotUpdateJudge(&message, &linked); err != nil {
						*errCh <- err
					}
				case "swap":
					if err := o.SwapUpdateJudge(&message, &linked); err != nil {
						*errCh <- err
					}
				}
			default:
				st := FormatingTimeStamp(message["T"].(float64))
				price, _ := decimal.NewFromString(message["p"].(string))
				size, _ := decimal.NewFromString(message["q"].(string))
				// is the buyer the mm
				var side string
				buyerIsMM := message["m"].(bool)
				if buyerIsMM {
					side = "sell"
				} else {
					side = "buy"
				}
				o.LocateTradeImpact(side, price, size, st)
				o.RenewTradeImpact()
			}
		}
	}
}

func (o *OrderBookBranch) LocateTradeImpact(side string, price, size decimal.Decimal, st time.Time) {
	switch side {
	case "buy":
		o.BuyTrade.mux.Lock()
		defer o.BuyTrade.mux.Unlock()
		o.BuyTrade.Qty = append(o.BuyTrade.Qty, size)
		o.BuyTrade.Stamp = append(o.BuyTrade.Stamp, st)
		o.BuyTrade.Notional = append(o.BuyTrade.Notional, price.Mul(size))
	case "sell":
		o.SellTrade.mux.Lock()
		defer o.SellTrade.mux.Unlock()
		o.SellTrade.Qty = append(o.SellTrade.Qty, size)
		o.SellTrade.Stamp = append(o.SellTrade.Stamp, st)
		o.SellTrade.Notional = append(o.SellTrade.Notional, price.Mul(size))
	}
}

func (o *OrderBookBranch) RenewTradeImpact() {
	var wg sync.WaitGroup
	wg.Add(2)
	now := time.Now()
	go func() {
		defer wg.Done()
		o.BuyTrade.mux.Lock()
		defer o.BuyTrade.mux.Unlock()
		var loc int = -1
		for i, st := range o.BuyTrade.Stamp {
			if !now.After(st.Add(o.LookBack)) {
				break
			}
			loc = i
		}
		if loc == -1 {
			return
		}
		o.BuyTrade.Stamp = o.BuyTrade.Stamp[loc+1:]
		o.BuyTrade.Qty = o.BuyTrade.Qty[loc+1:]
		o.BuyTrade.Notional = o.BuyTrade.Notional[loc+1:]

	}()
	go func() {
		defer wg.Done()
		o.SellTrade.mux.Lock()
		defer o.SellTrade.mux.Unlock()
		var loc int = -1
		for i, st := range o.SellTrade.Stamp {
			if !now.After(st.Add(o.LookBack)) {
				break
			}
			loc = i
		}
		if loc == -1 {
			return
		}
		o.SellTrade.Stamp = o.SellTrade.Stamp[loc+1:]
		o.SellTrade.Qty = o.SellTrade.Qty[loc+1:]
		o.SellTrade.Notional = o.SellTrade.Notional[loc+1:]
	}()
	wg.Wait()
}

func (o *OrderBookBranch) SpotUpdateJudge(message *map[string]interface{}, linked *bool) error {
	headID := decimal.NewFromFloat((*message)["U"].(float64))
	tailID := decimal.NewFromFloat((*message)["u"].(float64))
	snapID := o.LastUpdatedId.Add(decimal.NewFromInt(1))
	if !(*linked) {
		//U <= lastUpdateId+1 AND u >= lastUpdateId+1.
		if headID.LessThanOrEqual(snapID) && tailID.GreaterThanOrEqual(snapID) {
			(*linked) = true
			o.UpdateNewComing(message)
			o.LastUpdatedId = tailID
		}
	} else {
		if headID.Equal(snapID) {
			o.UpdateNewComing(message)
			o.LastUpdatedId = tailID
		} else {
			return errors.New("refresh.")
		}
	}
	return nil
}

func (o *OrderBookBranch) SwapUpdateJudge(message *map[string]interface{}, linked *bool) error {
	headID := decimal.NewFromFloat((*message)["U"].(float64))
	tailID := decimal.NewFromFloat((*message)["u"].(float64))
	puID := decimal.NewFromFloat((*message)["pu"].(float64))
	snapID := o.LastUpdatedId
	if !(*linked) {
		// drop u is < lastUpdateId
		if tailID.LessThan(snapID) {
			return nil
		}
		// U <= lastUpdateId AND u >= lastUpdateId
		if headID.LessThanOrEqual(snapID) && tailID.GreaterThanOrEqual(snapID) {
			(*linked) = true
			o.UpdateNewComing(message)
			o.LastUpdatedId = tailID
		}
	} else {
		// new event's pu should be equal to the previous event's u
		if puID.Equal(snapID) {
			o.UpdateNewComing(message)
			o.LastUpdatedId = tailID
		} else {
			return errors.New("refresh.")
		}
	}
	return nil
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
	logger.Infof("Binance %s %s %s socket connected.\n", symbol, product, channel)
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
	event, ok := res["e"].(string)
	if !ok {
		return nil
	}
	switch event {
	case "depthUpdate":
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
	case "trade":
		*mainCh <- res
	case "aggTrade":
		*mainCh <- res
	}
	return nil
}

func (w *WS) OutBinanceErr() map[string]interface{} {
	w.OnErr = true
	m := make(map[string]interface{})
	return m
}

func FormatingTimeStamp(timeFloat float64) time.Time {
	t := time.Unix(int64(timeFloat/1000), 0)
	return t
}
