package bnnapi

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

type UserDataBranch struct {
	spotAccount        spotAccountBranch
	snapShoted         bool
	cancel             *context.CancelFunc
	HttpUpdateInterval int
}

type spotAccountBranch struct {
	sync.RWMutex
	Data        *SpotAccountResponse
	LastUpdated decimal.Decimal
}

func (u *UserDataBranch) Close() {
	(*u.cancel)()
}

// default is 900 sec
func (u *UserDataBranch) SetHttpUpdateInterval(input int) {
	u.HttpUpdateInterval = input
}

func (c *Client) SpotUserData(logger *log.Logger) *UserDataBranch {
	user := c.localUserData("spot", "", logger)
	return user
}

func (u *UserDataBranch) GetSpotAccount() *SpotAccountResponse {
	u.spotAccount.RLock()
	defer u.spotAccount.RUnlock()
	return u.spotAccount.Data
}

// if it's isomargin, should pass symbol. Else just pass ""
func (c *Client) localUserData(product, symbol string, logger *log.Logger) *UserDataBranch {
	var u UserDataBranch
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = &cancel
	u.HttpUpdateInterval = 900
	userData := make(chan map[string]interface{}, 100)
	errCh := make(chan error, 5)
	// stream user data
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				res, err := c.getListenKeyHub(product, symbol) // delete listen key
				if err != nil {
					log.Println("retry listen key for user data stream in 5 sec..")
					time.Sleep(time.Second * 5)
					continue
				}
				if err := BNNUserData(ctx, product, res.ListenKey, logger, &userData); err == nil {
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
				switch product {
				case "spot":
					u.maintainSpotUserData(ctx, c, &userData, &errCh)
				}
				logger.Warningf("Refreshing %s local user data.\n", product)
				time.Sleep(time.Second)
			}
		}
	}()
	return &u
}

func (u *UserDataBranch) getSpotAccountSnapShot(client *Client, errCh *chan error) {
	u.spotAccount.Lock()
	defer u.spotAccount.Unlock()
	res, err := client.SpotAccount()
	if err != nil {
		*errCh <- err
		return
	}
	u.spotAccount.Data = res
	u.snapShoted = true
	u.spotAccount.LastUpdated = decimal.NewFromInt(int64(res.UpdateTime))
}

func (u *UserDataBranch) maintainSpotUserData(
	ctx context.Context,
	client *Client,
	userData *chan map[string]interface{},
	errCh *chan error,
) {
	u.snapShoted = false
	u.spotAccount.LastUpdated = decimal.NewFromInt(0)
	// get the first snapshot to initial data struct
	go func() {
		time.Sleep(time.Second)
		u.getSpotAccountSnapShot(client, errCh)
	}()
	// update snapshot with steady interval
	go func() {
		snap := time.NewTicker(time.Second * time.Duration(u.HttpUpdateInterval))
		defer snap.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-snap.C:
				u.snapShoted = false
				u.getSpotAccountSnapShot(client, errCh)
			default:
				time.Sleep(time.Second)
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-(*errCh):
			return
		default:
			if !u.snapShoted {
				time.Sleep(time.Second)
				continue
			}
			message := <-(*userData)
			event, ok := message["e"].(string)
			if !ok {
				continue
			}
			switch event {
			case "outboundAccountPosition":
				u.updateSpotAccountData(&message)
			case "balanceUpdate":
				// next stage, no use for now
			default:
				// pass
			}
		}
	}
}

func (u *UserDataBranch) updateSpotAccountData(message *map[string]interface{}) {
	array, ok := (*message)["B"].([]interface{})
	if !ok {
		return
	}
	u.spotAccount.Lock()
	defer u.spotAccount.Unlock()
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
		for idx, bal := range u.spotAccount.Data.Balances {
			if bal.Asset == asset {
				u.spotAccount.Data.Balances[idx].Free = free
				u.spotAccount.Data.Balances[idx].Free = lock
				return
			}
		}
	}
}

func BNNUserData(ctx context.Context, product, listenKey string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w wS
	var duration time.Duration = 600
	w.Logger = logger
	w.OnErr = false
	var buffer bytes.Buffer
	switch product {
	case "swap":
		buffer.WriteString("wss://fstream3.binance.com/ws/")
	default:
		buffer.WriteString("wss://stream.binance.com:9443/ws/")
	}
	buffer.WriteString(listenKey)
	url := buffer.String()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	log.Println("Connected:", url)
	w.Conn = conn
	defer conn.Close()
	if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			w.OutBinanceErr()
			message := "Binance User Data closed..."
			log.Println(message)
			return errors.New(message)
		default:
			if w.Conn == nil {
				w.OutBinanceErr()
				message := "Binance User Data reconnect..."
				log.Println(message)
				return errors.New(message)
			}
			_, buf, err := conn.ReadMessage()
			if err != nil {
				w.OutBinanceErr()
				message := "Binance User Data reconnect..."
				log.Println(message)
				return errors.New(message)
			}
			res, err1 := DecodingMap(buf, logger)
			if err1 != nil {
				w.OutBinanceErr()
				message := "Binance User Data reconnect..."
				log.Println(message, err1)
				return err1
			}
			// insert to chan
			*mainCh <- res
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
		}
	}
}

func (c *Client) getListenKeyHub(product, symbol string) (*ListenKeyResponse, error) {
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
