package bnnapi

import (
	"net/http"
)

type DepthOpts struct {
	Symbol string `url:"symbol"`
	Limit  int    `url:"limit"`
}

// Valid limits:[5, 10, 20, 50, 100, 500, 1000, 5000]
func (b *Client) SpotDepth(symbol string, limit int) (*Depth, error) {
	opts := DepthOpts{
		Symbol: symbol,
		Limit:  limit,
	}
	if opts.Limit == 0 || opts.Limit > 5000 {
		opts.Limit = 5000
	}
	res, err := b.do("spot", http.MethodGet, "api/v3/depth", opts, false, false)
	if err != nil {
		b.Logger.Println(err)
		return nil, err
	}
	depth := &Depth{}
	err = json.Unmarshal(res, &depth)
	if err != nil {
		b.Logger.Println(err)
		return nil, err
	}
	return depth, nil
}

type Depth struct {
	LastUpdateID int           `json:"lastUpdateId"`
	Bids         []interface{} `json:"bids"`
	Asks         []interface{} `json:"asks"`
}

func (b *Client) SwapDepth(symbol string, limit int) (*SwapDepth, error) {
	opts := DepthOpts{
		Symbol: symbol,
		Limit:  limit,
	}
	if opts.Limit == 0 || opts.Limit > 100 {
		opts.Limit = 100
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/depth", opts, false, false)
	if err != nil {
		b.Logger.Println(err)
		return nil, err
	}
	depth := &SwapDepth{}
	err = json.Unmarshal(res, &depth)
	if err != nil {
		b.Logger.Println(err)
		return nil, err
	}
	return depth, nil
}

type SwapDepth struct {
	LastUpdateID    int           `json:"lastUpdateId"`
	Bids            []interface{} `json:"bids"`
	Asks            []interface{} `json:"asks"`
	MessageOutTime  int64         `json:"E"`
	TransactionTime int64         `json:"T"`
}
