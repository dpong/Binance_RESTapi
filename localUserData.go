package bnnapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

type SpotUserDataBranch struct {
	spotAccount        spotAccountBranch
	cancel             *context.CancelFunc
	HttpUpdateInterval int
}

type spotAccountBranch struct {
	sync.RWMutex
	Data           *SpotAccountResponse
	LastUpdated    time.Time
	spotsnapShoted bool
}

func (u *SpotUserDataBranch) Close() {
	(*u.cancel)()
}

// default is 60 sec
func (u *SpotUserDataBranch) SetHttpUpdateInterval(input int) {
	u.HttpUpdateInterval = input
}

func (c *Client) SpotUserData(logger *log.Logger) *SpotUserDataBranch {
	user := c.localUserData("spot", "", logger)
	return user
}

func (u *SpotUserDataBranch) SpotAccount() *SpotAccountResponse {
	u.spotAccount.RLock()
	defer u.spotAccount.RUnlock()
	return u.spotAccount.Data
}

// if it's isomargin, should pass symbol. Else just pass ""
func (c *Client) localUserData(product, symbol string, logger *log.Logger) *SpotUserDataBranch {
	var u SpotUserDataBranch
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = &cancel
	u.HttpUpdateInterval = 60
	userData := make(chan map[string]interface{}, 100)
	errCh := make(chan error, 5)
	// stream user data
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				res, err := c.GetListenKeyHub(product, symbol) // delete listen key
				if err != nil {
					log.Println("retry listen key for user data stream in 5 sec..")
					time.Sleep(time.Second * 5)
					continue
				}
				if err := c.bNNUserData(ctx, product, res.ListenKey, logger, &userData); err == nil {
					return
				}
				errCh <- errors.New("Reconnect websocket")
				time.Sleep(time.Second)
			}
		}
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
				if err := u.getSpotAccountSnapShot(c); err != nil {
					message := fmt.Sprintf("fail to getSpotAccountSnapShot() with err: %s", err.Error())
					logger.Errorln(message)
				}

			default:
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
	// wait for connecting
	time.Sleep(time.Second * 5)
	return &u
}

func (u *SpotUserDataBranch) getSpotAccountSnapShot(client *Client) error {
	u.spotAccount.Lock()
	defer u.spotAccount.Unlock()
	u.spotAccount.spotsnapShoted = false
	res, err := client.SpotAccount()
	if err != nil {
		return err
	}
	u.spotAccount.Data = res
	u.spotAccount.spotsnapShoted = true
	u.spotAccount.LastUpdated = time.Now()
	return nil
}

func (u *SpotUserDataBranch) maintainSpotUserData(
	ctx context.Context,
	client *Client,
	userData *chan map[string]interface{},
	errCh *chan error,
) {
	innerErr := make(chan error, 1)
	// get the first snapshot to initial data struct
	if err := u.getSpotAccountSnapShot(client); err != nil {
		return
	}
	// self check
	go func() {
		check := time.NewTicker(time.Second * 10)
		defer check.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-innerErr:
				return
			case <-check.C:
				last := u.lastUpdateTime()
				if time.Now().After(last.Add(time.Second * time.Duration(u.HttpUpdateInterval))) {
					*errCh <- errors.New("spot user data out of date")
					return
				}
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
			innerErr <- errors.New("restart")
			return
		default:
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

func (u *SpotUserDataBranch) lastUpdateTime() time.Time {
	u.spotAccount.RLock()
	defer u.spotAccount.RUnlock()
	return u.spotAccount.LastUpdated
}

func (u *SpotUserDataBranch) updateSpotAccountData(message *map[string]interface{}) {
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
				u.spotAccount.Data.Balances[idx].Locked = lock
				u.spotAccount.LastUpdated = time.Now()
				return
			}
		}
	}
}

func (c *Client) bNNUserData(ctx context.Context, product, listenKey string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w wS
	var duration time.Duration = 1810
	w.Logger = logger
	w.OnErr = false
	var buffer bytes.Buffer
	innerErr := make(chan error, 1)
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
				if err := c.PutListenKeyHub(product, listenKey); err != nil {
					// time out in 1 sec
					w.Conn.SetReadDeadline(time.Now().Add(time.Second))
					return
				}
				w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration))
			default:
				time.Sleep(time.Second)
			}
		}
	}()
	read := time.NewTicker(time.Millisecond * 50)
	defer read.Stop()
	for {
		select {
		case <-ctx.Done():
			w.OutBinanceErr()
			message := "Binance User Data closed..."
			log.Println(message)
			return errors.New(message)
		case <-read.C:
			if w.Conn == nil {
				w.OutBinanceErr()
				message := "Binance User Data reconnect..."
				log.Println(message)
				innerErr <- errors.New("restart")
				return errors.New(message)
			}
			_, buf, err := conn.ReadMessage()
			if err != nil {
				w.OutBinanceErr()
				message := "Binance User Data reconnect..."
				log.Println(message)
				innerErr <- errors.New("restart")
				return errors.New(message)
			}
			res, err1 := DecodingMap(buf, logger)
			if err1 != nil {
				w.OutBinanceErr()
				message := "Binance User Data reconnect..."
				log.Println(message, err1)
				innerErr <- errors.New("restart")
				return err1
			}
			// insert to chan
			*mainCh <- res
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				innerErr <- errors.New("restart")
				return err
			}
		default:
			time.Sleep(time.Millisecond * 10)
		}
	}
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
