package bnnapi

import "net/http"

type SymbolPriceOpts struct {
	Symbol string `url:"symbol"`
}

func (b *Client) SpotPrice(symbol string) (*SymbolPrice, error) {
	opts := SymbolPriceOpts{
		Symbol: symbol,
	}
	res, err := b.do("spot", http.MethodGet, "api/v3/ticker/price", opts, false, false)
	if err != nil {
		return nil, err
	}

	price := &SymbolPrice{}
	if err = json.Unmarshal(res, &price); err != nil {
		return nil, err
	}
	return price, nil
}

func (b *Client) SpotPrices() ([]*SymbolPrice, error) {
	res, err := b.do("spot", http.MethodGet, "api/v3/ticker/price", nil, false, false)
	if err != nil {
		return nil, err
	}
	prices := []*SymbolPrice{}
	err = json.Unmarshal(res, &prices)
	if err != nil {
		return []*SymbolPrice{}, err
	}
	return prices, nil
}

func (b *Client) SpotOrderBookTickers() ([]*SymbolOrderBookTicker, error) {
	res, err := b.do("spot", http.MethodGet, "api/v3/ticker/bookTicker", nil, false, false)
	if err != nil {
		return nil, err
	}
	prices := []*SymbolOrderBookTicker{}
	err = json.Unmarshal(res, &prices)
	if err != nil {
		return []*SymbolOrderBookTicker{}, err
	}
	return prices, nil
}

type SymbolPrice struct {
	Symbol string
	Price  string
}

type SymbolOrderBookTicker struct {
	Symbol   string `json:"symbol"`
	BidPrice string `json:"bidPrice"`
	BidQty   string `json:"bidQty"`
	AskPrice string `json:"askPrice"`
	AskQty   string `json:"askQty"`
	Time     int64  `json:"time, omitempty"`
}

func (b *Client) SwapPrices() ([]*SwapPrice, error) {
	res, err := b.do("future", http.MethodGet, "fapi/v1/ticker/price", nil, false, false)
	if err != nil {
		return nil, err
	}
	prices := []*SwapPrice{}
	err = json.Unmarshal(res, &prices)
	if err != nil {
		return []*SwapPrice{}, err
	}
	return prices, nil
}

func (b *Client) SwapOrderBookTickers() ([]*SymbolOrderBookTicker, error) {
	res, err := b.do("future", http.MethodGet, "fapi/v1/ticker/bookTicker", nil, false, false)
	if err != nil {
		return nil, err
	}
	prices := []*SymbolOrderBookTicker{}
	err = json.Unmarshal(res, &prices)
	if err != nil {
		return []*SymbolOrderBookTicker{}, err
	}
	return prices, nil
}

type SwapPrice struct {
	Symbol string
	Price  string
	Time   int64
}

func (b *Client) SwapMarkPrices(symbol string) ([]*SwapMarkPriceResponse, error) {
	opts := SwapMarkPriceOpts{}
	if symbol != "" {
		opts.Symbol = symbol
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/premiumIndex", opts, false, false)
	if err != nil {
		return nil, err
	}
	prices := []*SwapMarkPriceResponse{}
	err = json.Unmarshal(res, &prices)
	if err != nil {
		return []*SwapMarkPriceResponse{}, err
	}
	return prices, nil
}

type SwapMarkPriceOpts struct {
	Symbol string `url:"symbol"`
}

type SwapMarkPriceResponse struct {
	Symbol          string `json:"symbol"`
	MarkPrice       string `json:"markPrice"`
	IndexPrice      string `json:"indexPrice"`
	LastFundingRate string `json:"lastFundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
	Time            int64  `json:"time"`
}
