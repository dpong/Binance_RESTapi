package bnnapi

import (
	"net/http"
	"strings"
)

func (b *Client) SpotPlaceOrder(symbol, side string, price, size float64, orderType, timeInforce string) (*SpotOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	utype := strings.ToUpper(orderType)
	var utif string
	if timeInforce == "" {
		utif = "GTC"
	} else {
		utif = strings.ToUpper(timeInforce)
	}
	opts := PlaceOrderOpts{
		Symbol:      usymbol,
		Side:        uside,
		Price:       price,
		Qty:         size,
		Type:        utype,
		TimeInForce: utif,
	}
	res, err := b.do("spot", http.MethodPost, "api/v3/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &SpotOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) SpotPlaceOrderMarket(symbol, side string, size float64, clientID string) (*SpotOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	opts := PlaceOrderOptsMarket{
		Symbol: usymbol,
		Side:   uside,
		Qty:    size,
		Type:   "market",
	}
	res, err := b.do("spot", http.MethodPost, "api/v3/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &SpotOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type PlaceOrderOpts struct {
	Symbol      string  `url:"symbol"`
	Price       float64 `url:"price"`
	Qty         float64 `url:"quantity"`
	TimeInForce string  `url:"timeInForce"`
	Type        string  `url:"type"`
	Side        string  `url:"side"`
}

type PlaceOrderOptsMarket struct {
	Symbol   string  `url:"symbol"`
	Qty      float64 `url:"quantity"`
	Type     string  `url:"type"`
	Side     string  `url:"side"`
	ClientID string  `url:"newClientOrderId, omitempty"`
}

type SpotOrderResponse struct {
	Symbol              string `json:"symbol"`
	OrderID             int    `json:"orderId"`
	OrderListID         int    `json:"orderListId, omitempty"`
	ClientOrderID       string `json:"clientOrderId"`
	TransactTime        int64  `json:"transactTime"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	Fills               []struct {
		Price           string `json:"price"`
		Qty             string `json:"qty"`
		Commission      string `json:"commission"`
		CommissionAsset string `json:"commissionAsset"`
	} `json:"fills, omitempty"`
}

func (b *Client) SpotCancelOrder(symbol string, oid int) (*SpotCancelOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := SpotOIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("spot", http.MethodDelete, "api/v3/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &SpotCancelOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type SpotOIDOpts struct {
	Symbol string `url:"symbol"`
	Oid    int    `url:"orderId"`
}

type OIDOpts struct {
	Symbol   string `url:"symbol"`
	Oid      int    `url:"orderId"`
	Isolated string `url:"isIsolated, omitempty"`
}

type SpotCancelOrderResponse struct {
	Symbol              string `json:"symbol"`
	OrigClientOrderID   string `json:"origClientOrderId"`
	OrderID             int    `json:"orderId"`
	OrderListID         int    `json:"orderListId"`
	ClientOrderID       string `json:"clientOrderId"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
}

type SpotQueryOrderResponse struct {
	Symbol              string `json:"symbol"`
	OrderID             int    `json:"orderId"`
	OrderListID         int    `json:"orderListId"`
	ClientOrderID       string `json:"clientOrderId"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	StopPrice           string `json:"stopPrice"`
	IcebergQty          string `json:"icebergQty"`
	Time                int64  `json:"time"`
	UpdateTime          int64  `json:"updateTime"`
	IsWorking           bool   `json:"isWorking"`
	OrigQuoteOrderQty   string `json:"origQuoteOrderQty"`
}

func (b *Client) SpotQueryOrder(symbol string, oid int) (*SpotQueryOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := SpotOIDOpts{
		Symbol: usymbol,
		Oid:    oid,
	}
	res, err := b.do("spot", http.MethodGet, "api/v3/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &SpotQueryOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
