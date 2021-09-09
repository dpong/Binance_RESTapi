package bnnapi

import (
	"net/http"
)

func (b *Client) FundingRateHistory(symbol string, limit int, start, end int64) ([]*FundingData, error) {
	opts := FundingRateOpts{
		Symbol: symbol,
		Limit:  limit,
	}
	if start != 0 && end != 0 {
		opts.StartTime = start
		opts.EndTime = end
	}
	if opts.Limit == 0 || opts.Limit > 1000 {
		opts.Limit = 1000
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/fundingRate", opts, false, false)
	if err != nil {
		return nil, err
	}
	funding := []*FundingData{}
	err = json.Unmarshal(res, &funding)
	if err != nil {
		return nil, err
	}
	return funding, nil
}

type FundingRateOpts struct {
	Symbol    string `url:"symbol"`
	Limit     int    `url:"limit"`
	StartTime int64  `url:"startTime,omitempty"`
	EndTime   int64  `url:"endTime,omitempty"`
}

type FundingData struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	FundingTime int64  `json:"fundingTime"`
}

func (b *Client) MarkPrice() ([]*MarkPriceData, error) {
	res, err := b.do("future", http.MethodGet, "fapi/v1/premiumIndex", nil, false, false)
	if err != nil {
		return nil, err
	}
	price := []*MarkPriceData{}
	err = json.Unmarshal(res, &price)
	if err != nil {
		return nil, err
	}
	return price, nil
}

type MarkPriceData struct {
	Symbol          string `json:"symbol"`
	MarkPrice       string `json:"markPrice"`
	LastFundingRate string `json:"lastFundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
	Time            int64  `json:"time"`
}

func (b *Client) GetIncomeHistory(method, symbol string, limit int, start, end int64) ([]*IncomeResponse, error) {
	opts := IncomeHisOpts{
		Symbol:     symbol,
		Limit:      limit,
		IncomeType: method,
	}
	if start != 0 && end != 0 {
		opts.StartTime = start
		opts.EndTime = end
	}
	if opts.Limit == 0 || opts.Limit > 1000 {
		opts.Limit = 1000
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/income", opts, true, false)
	if err != nil {
		return nil, err
	}
	income := []*IncomeResponse{}
	err = json.Unmarshal(res, &income)
	if err != nil {
		return nil, err
	}
	return income, nil
}

type IncomeHisOpts struct {
	Symbol     string `url:"symbol"`
	IncomeType string `url:"incomeType"`
	StartTime  int64  `url:"startTime,omitempty"`
	EndTime    int64  `url:"endTime,omitempty"`
	Limit      int    `url:"limit"`
}

type IncomeResponse struct {
	Symbol     string `json:"symbol"`
	IncomeType string `json:"incomeType"`
	Income     string `json:"income"`
	Asset      string `json:"asset"`
	Info       string `json:"info"`
	Time       int64  `json:"time"`
	TranID     int    `json:"tranId"`
	TradeID    string `json:"tradeId"`
}
