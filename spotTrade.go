package bnnapi

import (
	"net/http"
	"strings"
)

func (b *Client) SpotPlaceOrder(symbol, side string, price, size string, orderType, timeInforce string) (*SpotOrderResponse, error) {
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

func (b *Client) SpotPlaceOrderMarket(symbol, side string, size string, clientID string) (*SpotOrderResponse, error) {
	usymbol := strings.ToUpper(symbol)
	uside := strings.ToUpper(side)
	opts := PlaceOrderOptsMarket{
		Symbol: usymbol,
		Side:   uside,
		Qty:    size,
		Type:   "MARKET",
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
	Symbol      string `url:"symbol"`
	Price       string `url:"price"`
	Qty         string `url:"quantity"`
	TimeInForce string `url:"timeInForce"`
	Type        string `url:"type"`
	Side        string `url:"side"`
}

type PlaceOrderOptsMarket struct {
	Symbol   string `url:"symbol"`
	Qty      string `url:"quantity"`
	Type     string `url:"type"`
	Side     string `url:"side"`
	ClientID string `url:"newClientOrderId,omitempty"`
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

type CancelAllSpotOrdersResponse []struct {
	Symbol              string `json:"symbol"`
	Origclientorderid   string `json:"origClientOrderId,omitempty"`
	Orderid             int    `json:"orderId,omitempty"`
	Orderlistid         int    `json:"orderListId"`
	Clientorderid       string `json:"clientOrderId,omitempty"`
	Price               string `json:"price,omitempty"`
	Origqty             string `json:"origQty,omitempty"`
	Executedqty         string `json:"executedQty,omitempty"`
	Cummulativequoteqty string `json:"cummulativeQuoteQty,omitempty"`
	Status              string `json:"status,omitempty"`
	Timeinforce         string `json:"timeInForce,omitempty"`
	Type                string `json:"type,omitempty"`
	Side                string `json:"side,omitempty"`
	Contingencytype     string `json:"contingencyType,omitempty"`
	Liststatustype      string `json:"listStatusType,omitempty"`
	Listorderstatus     string `json:"listOrderStatus,omitempty"`
	Listclientorderid   string `json:"listClientOrderId,omitempty"`
	Transactiontime     int64  `json:"transactionTime,omitempty"`
	Orders              []struct {
		Symbol        string `json:"symbol"`
		Orderid       int    `json:"orderId"`
		Clientorderid string `json:"clientOrderId"`
	} `json:"orders,omitempty"`
	Orderreports []struct {
		Symbol              string `json:"symbol"`
		Origclientorderid   string `json:"origClientOrderId"`
		Orderid             int    `json:"orderId"`
		Orderlistid         int    `json:"orderListId"`
		Clientorderid       string `json:"clientOrderId"`
		Price               string `json:"price"`
		Origqty             string `json:"origQty"`
		Executedqty         string `json:"executedQty"`
		Cummulativequoteqty string `json:"cummulativeQuoteQty"`
		Status              string `json:"status"`
		Timeinforce         string `json:"timeInForce"`
		Type                string `json:"type"`
		Side                string `json:"side"`
		Stopprice           string `json:"stopPrice,omitempty"`
		Icebergqty          string `json:"icebergQty"`
	} `json:"orderReports,omitempty"`
}

type OnlySymbolOpt struct {
	Symbol string `url:"symbol"`
}

func (b *Client) CancelAllSpotOrders(symbol string) (*CancelAllSpotOrdersResponse, error) {
	usymbol := strings.ToUpper(symbol)
	opts := OnlySymbolOpt{
		Symbol: usymbol,
	}
	res, err := b.do("spot", http.MethodDelete, "api/v3/openOrders", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &CancelAllSpotOrdersResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type SpotCurrentOpenOrdersResponse struct {
	Symbol              string `json:"symbol"`
	Orderid             int    `json:"orderId"`
	Orderlistid         int    `json:"orderListId"`
	Clientorderid       string `json:"clientOrderId"`
	Price               string `json:"price"`
	Origqty             string `json:"origQty"`
	Executedqty         string `json:"executedQty"`
	Cummulativequoteqty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	Timeinforce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	Stopprice           string `json:"stopPrice"`
	Icebergqty          string `json:"icebergQty"`
	Time                int64  `json:"time"`
	Updatetime          int64  `json:"updateTime"`
	Isworking           bool   `json:"isWorking"`
	Origquoteorderqty   string `json:"origQuoteOrderQty"`
}

func (b *Client) GetCurrentSpotOrders(symbol string) ([]SpotCurrentOpenOrdersResponse, error) {
	var output []byte
	if symbol == "" {
		res, err := b.do("spot", http.MethodGet, "api/v3/openOrders", nil, true, false)
		if err != nil {
			return nil, err
		}
		output = res
	} else {
		usymbol := strings.ToUpper(symbol)
		opts := OnlySymbolOpt{
			Symbol: usymbol,
		}
		res, err := b.do("spot", http.MethodGet, "api/v3/openOrders", opts, true, false)
		if err != nil {
			return nil, err
		}
		output = res
	}
	resp := &[]SpotCurrentOpenOrdersResponse{}
	err := json.Unmarshal(output, resp)
	if err != nil {
		return nil, err
	}
	return *resp, nil
}
