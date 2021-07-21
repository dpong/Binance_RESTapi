package bnnapi

import (
	"net/http"
	"strings"
)

type PlaceOrderOptsFuture struct {
	Symbol      string  `url:"symbol"`
	Price       float64 `url:"price"`
	Qty         float64 `url:"quantity"`
	TimeInForce string  `url:"timeInForce"`
	Type        string  `url:"type"`
	Side        string  `url:"side"`
	ReduceOnly  string  `url:"reduceOnly"`
}

type PlaceOrderOptsFutureMarket struct {
	Symbol     string  `url:"symbol"`
	Qty        float64 `url:"quantity"`
	Type       string  `url:"type"`
	Side       string  `url:"side"`
	ReduceOnly string  `url:"reduceOnly"`
	ClientID   string  `url:"newClientOrderId, omitempty"`
}

func (b *Client) FuturePlaceOrderMarket(symbol, side string, size float64, reduceOnly, clientID string) (*FutureOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	opts := PlaceOrderOptsFutureMarket{
		Symbol:     usymbol,
		Side:       uside,
		Qty:        size,
		Type:       "market",
		ReduceOnly: reduceOnly,
	}
	if clientID != "" {
		opts.ClientID = clientID
	}
	res, err := b.do("future", http.MethodPost, "fapi/v1/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &FutureOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) FuturePlaceOrder(symbol, side string, price, size float64, orderType, timeInforce, reduceOnly string) (*FutureOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	utype := strings.ToUpper(orderType)
	var utif string
	if timeInforce == "" {
		utif = "GTC"
	} else {
		utif = strings.ToUpper(timeInforce)
	}
	opts := PlaceOrderOptsFuture{
		Symbol:      usymbol,
		Side:        uside,
		Price:       price,
		Qty:         size,
		Type:        utype,
		TimeInForce: utif,
		ReduceOnly:  reduceOnly,
	}
	res, err := b.do("future", http.MethodPost, "fapi/v1/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &FutureOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type FutureOrderResponse struct {
	ClientOrderID string `json:"clientOrderId"`
	CumQty        string `json:"cumQty"`
	CumQuote      string `json:"cumQuote"`
	ExecutedQty   string `json:"executedQty"`
	OrderID       int    `json:"orderId"`
	AvgPrice      string `json:"avgPrice, omitempty"`
	OrigQty       string `json:"origQty"`
	Price         string `json:"price"`
	ReduceOnly    bool   `json:"reduceOnly"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	Status        string `json:"status"`
	StopPrice     string `json:"stopPrice, omitempty"`
	ClosePosition bool   `json:"closePosition, omitempty"`
	Symbol        string `json:"symbol"`
	TimeInForce   string `json:"timeInForce"`
	Type          string `json:"type"`
	OrigType      string `json:"origType"`
	ActivatePrice string `json:"activatePrice, omitempty"`
	PriceRate     string `json:"priceRate, omitempty"`
	UpdateTime    int64  `json:"updateTime"`
	WorkingType   string `json:"workingType"`
}

func (b *Client) FutureCancelOrder(symbol string, oid int) (*FutureOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("future", http.MethodDelete, "fapi/v1/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &FutureOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) FutureOpenOrder(symbol string, oid int) (*FutureOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/openOrder", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &FutureOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type FutureQueryOrderResonse struct {
	AvgPrice      string `json:"avgPrice"`
	ClientOrderID string `json:"clientOrderId"`
	CumQuote      string `json:"cumQuote"`
	ExecutedQty   string `json:"executedQty"`
	OrderID       int    `json:"orderId"`
	OrigQty       string `json:"origQty"`
	OrigType      string `json:"origType"`
	Price         string `json:"price"`
	ReduceOnly    bool   `json:"reduceOnly"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	Status        string `json:"status"`
	StopPrice     string `json:"stopPrice"`
	ClosePosition bool   `json:"closePosition"`
	Symbol        string `json:"symbol"`
	Time          int64  `json:"time"`
	TimeInForce   string `json:"timeInForce"`
	Type          string `json:"type"`
	ActivatePrice string `json:"activatePrice"`
	PriceRate     string `json:"priceRate"`
	UpdateTime    int64  `json:"updateTime"`
	WorkingType   string `json:"workingType"`
	PriceProtect  bool   `json:"priceProtect"`
}

func (b *Client) FutureQueryOrder(symbol string, oid int) (*FutureQueryOrderResonse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("future", http.MethodGet, "/fapi/v1/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &FutureQueryOrderResonse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
