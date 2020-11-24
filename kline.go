package bnnapi

import (
	"fmt"
	"net/http"
	"time"
)

func (b *Client) SpotKlines(symbol, interval string, limit int, start, end time.Time) ([][]interface{}, error) {
	opts := KlinesOpts{
		Symbol:   symbol,
		Interval: interval,
		Limit:    limit,
	}
	if start.IsZero() == false && end.IsZero() == false {
		opts.StartTime = start.UnixNano() / int64(time.Millisecond)
		opts.EndTime = end.UnixNano() / int64(time.Millisecond)
	}
	if opts.Symbol == "" || opts.Interval == "" {
		return nil, fmt.Errorf("symbol or interval are missing")
	}
	if opts.Limit == 0 || opts.Limit > 1000 {
		opts.Limit = 1000
	}
	res, err := b.do("spot", http.MethodGet, "api/v3/klines", opts, false, false)
	if err != nil {
		b.Logger.Println(err)
		return nil, err
	}
	var klines [][]interface{}
	err = json.Unmarshal(res, &klines)
	if err != nil {
		b.Logger.Println(err)
		return nil, err
	}
	return klines, nil

}

func (b *Client) SwapKlines(symbol, interval string, limit int, start, end time.Time) ([][]interface{}, error) {
	opts := KlinesOpts{
		Symbol:   symbol,
		Interval: interval,
		Limit:    limit,
	}
	if start.IsZero() == false && end.IsZero() == false {
		opts.StartTime = start.UnixNano() / int64(time.Millisecond)
		opts.EndTime = end.UnixNano() / int64(time.Millisecond)
	}
	if opts.Symbol == "" || opts.Interval == "" {
		return nil, fmt.Errorf("symbol or interval are missing")
	}
	if opts.Limit == 0 || opts.Limit > 1000 {
		opts.Limit = 1000
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/klines", opts, false, false)
	if err != nil {
		b.Logger.Println(err)
		return nil, err
	}
	var klines [][]interface{}
	err = json.Unmarshal(res, &klines)
	if err != nil {
		b.Logger.Println(err)
		return nil, err
	}
	return klines, nil

}

type KlinesOpts struct {
	Symbol    string `url:"symbol"`   // Symbol is the symbol to fetch data for
	Interval  string `url:"interval"` // Interval is the interval for each kline/candlestick
	Limit     int    `url:"limit"`    // Limit is the maximal number of elements to receive. Max 500
	StartTime int64  `url:"startTime,omitempty"`
	EndTime   int64  `url:"endTime,omitempty"`
}

type Klines struct {
	OpenTime                 int64
	OpenPrice                string
	High                     string
	Low                      string
	ClosePrice               string
	Volume                   string
	CloseTime                int64
	QuoteAssetVolume         string
	Trades                   int
	TakerBuyBaseAssetVolume  string
	TakerBuyQuoteAssetVolume string
	Ignore                   string
}
