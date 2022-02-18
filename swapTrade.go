package bnnapi

import (
	"net/http"
	"strings"
)

type PlaceOrderOptsSwap struct {
	Symbol      string `url:"symbol"`
	Price       string `url:"price"`
	Qty         string `url:"quantity"`
	TimeInForce string `url:"timeInForce"`
	Type        string `url:"type"`
	Side        string `url:"side"`
	ReduceOnly  string `url:"reduceOnly"`
}

type PlaceOrderOptsSwapMarket struct {
	Symbol     string `url:"symbol"`
	Qty        string `url:"quantity"`
	Type       string `url:"type"`
	Side       string `url:"side"`
	ReduceOnly string `url:"reduceOnly"`
	ClientID   string `url:"newClientOrderId, omitempty"`
}

func (b *Client) SwapPlaceOrderMarket(symbol, side string, size string, reduceOnly, clientID string) (*FutureOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	opts := PlaceOrderOptsSwapMarket{
		Symbol:     usymbol,
		Side:       uside,
		Qty:        size,
		Type:       "MARKET",
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

func (b *Client) SwapPlaceOrder(symbol, side string, price, size string, orderType, timeInforce, reduceOnly string) (*FutureOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	utype := strings.ToUpper(orderType)
	var utif string
	if timeInforce == "" {
		utif = "GTC"
	} else {
		utif = strings.ToUpper(timeInforce)
	}
	opts := PlaceOrderOptsSwap{
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

type PlaceBatchOrdersOptsSwap struct {
	Orders []PlaceOrderOptsSwap `url:"batchOrders"`
}

func (b *Client) SwapPlaceBatchOrders(opts PlaceBatchOrdersOptsSwap) (*FutureBatchOrdersResponse, error) {
	res, err := b.do("future", http.MethodPost, "fapi/v1/batchOrders", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := FutureBatchOrdersResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
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

type FutureBatchOrdersResponse []struct {
	Clientorderid string `json:"clientOrderId,omitempty"`
	Cumqty        string `json:"cumQty,omitempty"`
	Cumquote      string `json:"cumQuote,omitempty"`
	Executedqty   string `json:"executedQty,omitempty"`
	Orderid       int    `json:"orderId,omitempty"`
	Avgprice      string `json:"avgPrice,omitempty"`
	Origqty       string `json:"origQty,omitempty"`
	Price         string `json:"price,omitempty"`
	Reduceonly    bool   `json:"reduceOnly,omitempty"`
	Side          string `json:"side,omitempty"`
	Positionside  string `json:"positionSide,omitempty"`
	Status        string `json:"status,omitempty"`
	Stopprice     string `json:"stopPrice,omitempty"`
	Symbol        string `json:"symbol,omitempty"`
	Timeinforce   string `json:"timeInForce,omitempty"`
	Type          string `json:"type,omitempty"`
	Origtype      string `json:"origType,omitempty"`
	Activateprice string `json:"activatePrice,omitempty"`
	Pricerate     string `json:"priceRate,omitempty"`
	Updatetime    int64  `json:"updateTime,omitempty"`
	Workingtype   string `json:"workingType,omitempty"`
	Priceprotect  bool   `json:"priceProtect,omitempty"`
	Code          int    `json:"code,omitempty"`
	Msg           string `json:"msg,omitempty"`
}

func (b *Client) SwapCancelOrder(symbol string, oid int) (*FutureOrderResponse, error) {
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

type OIDListOpts struct {
	Symbol string `url:"symbol"`
	Oid    []int  `url:"orderId"`
}

func (b *Client) CancelBatchOrders(symbol string, oids []int) (*FutureBatchOrdersResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OIDListOpts{
		Symbol: usymbol,
		Oid:    oids,
	}
	res, err := b.do("future", http.MethodDelete, "fapi/v1/batchOrders", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := FutureBatchOrdersResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (b *Client) SwapOpenOrder(symbol string, oid int) (*FutureOrderResponse, error) {
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

func (b *Client) SwapQueryOrder(symbol string, oid int) (*FutureQueryOrderResonse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/order", opts, true, false)
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

type SwapCurrentOpenOrdersResponse struct {
	Avgprice      string `json:"avgPrice"`
	Clientorderid string `json:"clientOrderId"`
	Cumquote      string `json:"cumQuote"`
	Executedqty   string `json:"executedQty"`
	Orderid       int    `json:"orderId"`
	Origqty       string `json:"origQty"`
	Origtype      string `json:"origType"`
	Price         string `json:"price"`
	Reduceonly    bool   `json:"reduceOnly"`
	Side          string `json:"side"`
	Positionside  string `json:"positionSide"`
	Status        string `json:"status"`
	Stopprice     string `json:"stopPrice"`
	Closeposition bool   `json:"closePosition"`
	Symbol        string `json:"symbol"`
	Time          int64  `json:"time"`
	Timeinforce   string `json:"timeInForce"`
	Type          string `json:"type"`
	Activateprice string `json:"activatePrice"`
	Pricerate     string `json:"priceRate"`
	Updatetime    int64  `json:"updateTime"`
	Workingtype   string `json:"workingType"`
	Priceprotect  bool   `json:"priceProtect"`
}

func (b *Client) GetCurrentSwapOrders(symbol string) ([]SwapCurrentOpenOrdersResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OnlySymbolOpt{
		Symbol: usymbol,
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/openOrders", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &[]SwapCurrentOpenOrdersResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return *resp, nil
}

type CancelAllSwaoOrdersResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (b *Client) CancelAllSwapOrders(symbol string) (*CancelAllSwaoOrdersResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OnlySymbolOpt{
		Symbol: usymbol,
	}
	res, err := b.do("future", http.MethodDelete, "fapi/v1/allOpenOrders", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &CancelAllSwaoOrdersResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
