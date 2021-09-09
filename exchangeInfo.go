package bnnapi

import "net/http"

func (b *Client) SpotInfo() (*SpotExchangeInfo, error) {
	res, err := b.do("spot", http.MethodGet, "api/v3/exchangeInfo", nil, false, false)
	if err != nil {
		return nil, err
	}
	exchange := &SpotExchangeInfo{}
	err = json.Unmarshal(res, &exchange)
	if err != nil {
		return nil, err
	}
	return exchange, nil

}

func (b *Client) SwapInfo() (*SwapExchangeInfo, error) {
	res, err := b.do("future", http.MethodGet, "fapi/v1/exchangeInfo", nil, false, false)
	if err != nil {
		return nil, err
	}
	exchange := &SwapExchangeInfo{}
	err = json.Unmarshal(res, &exchange)
	if err != nil {
		return nil, err
	}
	return exchange, nil

}

type SpotExchangeInfo struct {
	Timezone        string        `json:"timezone"`
	ServerTime      int64         `json:"serverTime"`
	RateLimits      []interface{} `json:"rateLimits"`
	ExchangeFilters []interface{} `json:"exchangeFilters"`
	Symbols         []struct {
		Symbol                 string        `json:"symbol"`
		Status                 string        `json:"status"`
		BaseAsset              string        `json:"baseAsset"`
		BaseAssetPrecision     int           `json:"baseAssetPrecision"`
		QuoteAsset             string        `json:"quoteAsset"`
		QuotePrecision         int           `json:"quotePrecision"`
		QuoteAssetPrecision    int           `json:"quoteAssetPrecision"`
		OrderTypes             []string      `json:"orderTypes"`
		IcebergAllowed         bool          `json:"icebergAllowed"`
		OcoAllowed             bool          `json:"ocoAllowed"`
		IsSpotTradingAllowed   bool          `json:"isSpotTradingAllowed"`
		IsMarginTradingAllowed bool          `json:"isMarginTradingAllowed"`
		Filters                []interface{} `json:"filters"`
		Permissions            []string      `json:"permissions"`
	} `json:"symbols"`
}

type SwapExchangeInfo struct {
	ExchangeFilters []interface{} `json:"exchangeFilters"`
	RateLimits      []struct {
		Interval      string `json:"interval"`
		IntervalNum   int    `json:"intervalNum"`
		Limit         int    `json:"limit"`
		RateLimitType string `json:"rateLimitType"`
	} `json:"rateLimits"`
	ServerTime int64 `json:"serverTime"`
	Symbols    []struct {
		Symbol                string `json:"symbol"`
		Status                string `json:"status"`
		MaintMarginPercent    string `json:"maintMarginPercent"`
		RequiredMarginPercent string `json:"requiredMarginPercent"`
		BaseAsset             string `json:"baseAsset"`
		QuoteAsset            string `json:"quoteAsset"`
		PricePrecision        int    `json:"pricePrecision"`
		QuantityPrecision     int    `json:"quantityPrecision"`
		BaseAssetPrecision    int    `json:"baseAssetPrecision"`
		QuotePrecision        int    `json:"quotePrecision"`
		Filters               []struct {
			MinPrice          string `json:"minPrice,omitempty"`
			MaxPrice          string `json:"maxPrice,omitempty"`
			FilterType        string `json:"filterType"`
			TickSize          string `json:"tickSize,omitempty"`
			StepSize          string `json:"stepSize,omitempty"`
			MaxQty            string `json:"maxQty,omitempty"`
			MinQty            string `json:"minQty,omitempty"`
			Limit             int    `json:"limit,omitempty"`
			MultiplierDown    string `json:"multiplierDown,omitempty"`
			MultiplierUp      string `json:"multiplierUp,omitempty"`
			MultiplierDecimal string `json:"multiplierDecimal,omitempty"`
		} `json:"filters"`
		OrderTypes  []string `json:"orderTypes"`
		TimeInForce []string `json:"timeInForce"`
	} `json:"symbols"`
	Timezone string `json:"timezone"`
}
