package bnnapi

import (
	"net/http"
	"strings"
)

func (b *Client) FuturePlaceOrder(symbol, side string, price, size float64, orderType, timeInforce string) (*FutureOrderResponse, error) {
	uside := strings.ToUpper(side)
	utype := strings.ToUpper(orderType)
	var utif string
	if timeInforce == "" {
		utif = "GTC"
	} else {
		utif = strings.ToUpper(timeInforce)
	}
	opts := PlaceOrderOpts{
		Symbol:      symbol,
		Side:        uside,
		Price:       price,
		Qty:         size,
		Type:        utype,
		TimeInForce: utif,
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
	opts := OIDOpts{
		Symbol: symbol,
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
	opts := OIDOpts{
		Symbol: symbol,
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
