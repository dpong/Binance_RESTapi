package bnnapi

import (
	"net/http"
	"strings"
)

type PlaceOrderOptsPerp struct {
	Symbol      string `url:"symbol"`
	Price       string `url:"price"`
	Qty         string `url:"quantity"`
	TimeInForce string `url:"timeInForce"`
	Type        string `url:"type"`
	Side        string `url:"side"`
	ReduceOnly  string `url:"reduceOnly"`
}

type PlaceOrderOptsPerpMarket struct {
	Symbol     string `url:"symbol"`
	Qty        string `url:"quantity"`
	Type       string `url:"type"`
	Side       string `url:"side"`
	ReduceOnly string `url:"reduceOnly"`
	ClientID   string `url:"newClientOrderId, omitempty"`
}

func (b *Client) PerpPlaceOrderMarket(symbol, side string, size string, reduceOnly, clientID string) (*PerpOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	opts := PlaceOrderOptsPerpMarket{
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
	resp := &PerpOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) PerpPlaceOrder(symbol, side string, price, size string, orderType, timeInforce, reduceOnly string) (*PerpOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	utype := strings.ToUpper(orderType)
	var utif string
	if timeInforce == "" {
		utif = "GTC"
	} else {
		utif = strings.ToUpper(timeInforce)
	}
	opts := PlaceOrderOptsPerp{
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
	resp := &PerpOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type PlaceBatchOrdersOptsPerp struct {
	OrderList string `url:"batchOrders"`
}

// max 5 orders per request
func (b *Client) PerpPlaceBatchOrders(orders []PlaceOrderOptsPerp) (*PerpBatchOrdersResponse, error) {
	opts := []map[string]interface{}{}
	for _, order := range orders {
		m := map[string]interface{}{}
		m["symbol"] = strings.ToUpper(order.Symbol)
		m["side"] = strings.ToUpper(order.Side)
		m["type"] = strings.ToUpper(order.Type)
		m["timeInForce"] = strings.ToUpper(order.TimeInForce)
		m["price"] = order.Price
		m["quantity"] = order.Qty
		m["reduceOnly"] = strings.ToLower(order.ReduceOnly)
		opts = append(opts, m)
	}
	out, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	input := PlaceBatchOrdersOptsPerp{}
	input.OrderList = Bytes2String(out)
	res, err := b.do("future", http.MethodPost, "fapi/v1/batchOrders", input, true, false)
	if err != nil {
		return nil, err
	}
	resp := PerpBatchOrdersResponse{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type PerpOrderResponse struct {
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

type PerpBatchOrdersResponse []struct {
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

func (b *Client) PerpCancelOrder(symbol string, oid int) (*PerpOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("future", http.MethodDelete, "fapi/v1/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &PerpOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type OIDListOpts struct {
	Symbol string `url:"symbol"`
	Oid    string `url:"orderIdList"`
}

// max 10 order per request
func (b *Client) PerpCancelBatchOrders(symbol string, oids []int) (*PerpBatchOrdersResponse, error) {
	out, err := json.Marshal(oids)
	if err != nil {
		return nil, err
	}
	opts := OIDListOpts{
		Symbol: strings.ToUpper(symbol),
		Oid:    Bytes2String(out),
	}
	res, err := b.do("future", http.MethodDelete, "fapi/v1/batchOrders", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := PerpBatchOrdersResponse{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (b *Client) PerpOpenOrder(symbol string, oid int) (*PerpOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/openOrder", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &PerpOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type PerpQueryOrderResonse struct {
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

func (b *Client) PerpQueryOrder(symbol string, oid int) (*PerpQueryOrderResonse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &PerpQueryOrderResonse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type PerpCurrentOpenOrdersResponse struct {
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

func (b *Client) GetCurrentPerpOrders(symbol string) ([]PerpCurrentOpenOrdersResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OnlySymbolOpt{
		Symbol: usymbol,
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/openOrders", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &[]PerpCurrentOpenOrdersResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return *resp, nil
}

type CancelAllPerpOrdersResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (b *Client) CancelAllPerpOrders(symbol string) (*CancelAllPerpOrdersResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OnlySymbolOpt{
		Symbol: usymbol,
	}
	res, err := b.do("future", http.MethodDelete, "fapi/v1/allOpenOrders", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &CancelAllPerpOrdersResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
