package bnnapi

import (
	"net/http"
	"strings"
)

type PlaceOrderOptsIsomargin struct {
	Symbol      string  `url:"symbol"`
	Price       float64 `url:"price"`
	Qty         float64 `url:"quantity"`
	TimeInForce string  `url:"timeInForce"`
	Type        string  `url:"type"`
	Side        string  `url:"side"`
	Isolated    string  `url:"isIsolated, omitempty"`
}

func (b *Client) MarginPlaceOrder(symbol, side string, price, size float64, orderType, timeInforce, isolated string) (*MarginOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	utype := strings.ToUpper(orderType)
	var utif string
	if timeInforce == "" {
		utif = "GTC"
	} else {
		utif = strings.ToUpper(timeInforce)
	}
	opts := PlaceOrderOptsIsomargin{
		Symbol:      usymbol,
		Side:        uside,
		Price:       price,
		Qty:         size,
		Type:        utype,
		Isolated:    isolated,
		TimeInForce: utif,
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/margin/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &MarginOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type MarginOrderResponse struct {
	Symbol                string `json:"symbol"`
	OrderID               int    `json:"orderId"`
	ClientOrderID         string `json:"clientOrderId"`
	TransactTime          int64  `json:"transactTime"`
	Price                 string `json:"price"`
	OrigQty               string `json:"origQty"`
	ExecutedQty           string `json:"executedQty"`
	CummulativeQuoteQty   string `json:"cummulativeQuoteQty"`
	Status                string `json:"status"`
	TimeInForce           string `json:"timeInForce"`
	Type                  string `json:"type"`
	Side                  string `json:"side"`
	MarginBuyBorrowAmount int    `json:"marginBuyBorrowAmount"`
	MarginBuyBorrowAsset  string `json:"marginBuyBorrowAsset"`
	IsIsolated            bool   `json:"isIsolated"`
	Fills                 []struct {
		Price           string `json:"price"`
		Qty             string `json:"qty"`
		Commission      string `json:"commission"`
		CommissionAsset string `json:"commissionAsset"`
	} `json:"fills"`
}

func (b *Client) MarginCancelOrder(symbol string, oid int, isolated string) (*MarginCancelOrderResponse, error) {
	opts := OIDOpts{
		Symbol:   symbol,
		Oid:      oid,
		Isolated: isolated,
	}
	res, err := b.do("spot", http.MethodDelete, "sapi/v1/margin/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &MarginCancelOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) MarginOpenOrder(symbol string, oid int, isolated string) (*MarginOpenOrderResponse, error) {
	opts := OIDOpts{
		Symbol:   symbol,
		Oid:      oid,
		Isolated: isolated,
	}
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/order", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &MarginOpenOrderResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type MarginCancelOrderResponse struct {
	Symbol              string `json:"symbol"`
	IsIsolated          bool   `json:"isIsolated"`
	OrderID             string `json:"orderId"`
	OrigClientOrderID   string `json:"origClientOrderId"`
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

type MarginOpenOrderResponse struct {
	ClientOrderID       string `json:"clientOrderId"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	ExecutedQty         string `json:"executedQty"`
	IcebergQty          string `json:"icebergQty"`
	IsWorking           bool   `json:"isWorking"`
	OrderID             int    `json:"orderId"`
	OrigQty             string `json:"origQty"`
	Price               string `json:"price"`
	Side                string `json:"side"`
	Status              string `json:"status"`
	StopPrice           string `json:"stopPrice"`
	Symbol              string `json:"symbol"`
	IsIsolated          bool   `json:"isIsolated"`
	Time                int64  `json:"time"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	UpdateTime          int64  `json:"updateTime"`
}
