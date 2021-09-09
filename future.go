package bnnapi

import (
	"errors"
	"net/http"
)

func (b *Client) SwapTransfer(method, asset string, amount float64) (*TransferResponse, error) {
	var transfertype int
	if method == "in" {
		transfertype = 1
	} else if method == "out" {
		transfertype = 2
	} else {
		return nil, errors.New("can't recognize margin account transfer method")
	}
	opts := TransferOpts{
		Asset:        asset,
		Amount:       amount,
		TransferType: transfertype,
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/futures/transfer", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &TransferResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) FutureTransfer(method, asset string, amount float64) (*TransferResponse, error) {
	var transfertype int
	if method == "in" {
		transfertype = 3
	} else if method == "out" {
		transfertype = 4
	} else {
		return nil, errors.New("can't recognize margin account transfer method")
	}
	opts := TransferOpts{
		Asset:        asset,
		Amount:       amount,
		TransferType: transfertype,
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/futures/transfer", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &TransferResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) SwapBalance() ([]*SwapBalanceResponse, error) {
	res, err := b.do("future", http.MethodGet, "fapi/v2/balance", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := []*SwapBalanceResponse{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type SwapBalanceResponse struct {
	AccountAlias       string `json:"accountAlias"`
	Asset              string `json:"asset`
	Balace             string `json:"balance"`
	CrossWalletBalance string `json:"crossWalletBalance"`
	AvailableBalance   string `json:"availableBalance"`
	MaxWithdrawAmount  string `json:"maxWithdrawAmount"`
}

func (b *Client) SwapAccount() (*SwapAccountResponse, error) {
	res, err := b.do("future", http.MethodGet, "fapi/v2/account", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := &SwapAccountResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) SwapPositions() ([]*SwapPositionResponse, error) {
	res, err := b.do("future", http.MethodGet, "fapi/v2/positionRisk", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := []*SwapPositionResponse{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type SwapAccountResponse struct {
	FeeTier                     int                  `json:"feeTier"`
	CanTrade                    bool                 `json:"canTrade"`
	CanDeposit                  bool                 `json:"canDeposit"`
	CanWithdraw                 bool                 `json:"canWithdraw"`
	UpdateTime                  int64                `json:"updateTime"`
	TotalInitialMargin          string               `json:"totalInitialMargin"`
	TotalMaintMargin            string               `json:"totalMaintMargin"`
	TotalWalletBalance          string               `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string               `json:"totalUnrealizedProfit"`
	TotalMarginBalance          string               `json:"totalMarginBalance"`
	TotalPositionInitialMargin  string               `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin string               `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     string               `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             string               `json:"totalCrossUnPnl"`
	AvailableBalance            string               `json:"availableBalance"`
	MaxWithdrawAmount           string               `json:"maxWithdrawAmount"`
	Assets                      []AssetsInAccount    `json:"assets"`
	Positions                   []PositionsInAccount `json:"positions"`
}

type AssetsInAccount struct {
	Asset                  string `json:"asset"`
	WalletBalance          string `json:"walletBalance"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	MarginBalance          string `json:"marginBalance"`
	MaintMargin            string `json:"maintMargin"`
	InitialMargin          string `json:"initialMargin"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	CrossWalletBalance     string `json:"crossWalletBalance"`
	CrossUnPnl             string `json:"crossUnPnl"`
	AvailableBalance       string `json:"availableBalance"`
	MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
}

type PositionsInAccount struct {
	Symbol                 string `json:"symbol"`
	InitialMargin          string `json:"initialMargin"`
	MaintMargin            string `json:"maintMargin"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	Leverage               string `json:"leverage"`
	Isolated               bool   `json:"isolated"`
	EntryPrice             string `json:"entryPrice"`
	MaxNotional            string `json:"maxNotional"`
	PositionSide           string `json:"positionSide"`
	PositionAmt            string `json:"positionAmt"`
}

type SwapPositionResponse struct {
	EntryPrice       string `json:"entryPrice"`
	MarginType       string `json:"marginType"`
	IsAutoAddMargin  string `json:"isAutoAddMargin"`
	IsolatedMargin   string `json:"isolatedMargin"`
	Leverage         string `json:"leverage"`
	LiquidationPrice string `json:"liquidationPrice"`
	MarkPrice        string `json:"markPrice"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	PositionAmt      string `json:"positionAmt"`
	Symbol           string `json:"symbol"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	PositionSide     string `json:"positionSide"`
}

func (b *Client) SwapOpenInterest(symbol string) (*SwapOpenInterestResponse, error) {
	opts := SwapOpenInterestOpts{
		Symbol: symbol,
	}
	res, err := b.do("future", http.MethodGet, "fapi/v1/openInterest", opts, false, false)
	if err != nil {
		return nil, err
	}
	resp := &SwapOpenInterestResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type SwapOpenInterestOpts struct {
	Symbol string `url:"symbol"`
}

type SwapOpenInterestResponse struct {
	OpenInterest string `json:"openInterest"`
	Symbol       string `json:"symbol"`
	Time         int64  `json:"time"`
}

func (b *Client) SwapNotionalandLeverage() (*[]NotionalandLeverage, error) {
	res, err := b.do("future", http.MethodGet, "/fapi/v1/leverageBracket", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := []NotionalandLeverage{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type NotionalandLeverage struct {
	Symbol   string     `json:"symbol"`
	Brackets []Brackets `json:"brackets"`
}

type Brackets struct {
	Bracket          int     `json:"bracket"`
	InitialLeverage  int     `json:"initialLeverage"`
	NotionalCap      int     `json:"notionalCap"`
	NotionalFloor    int     `json:"notionalFloor"`
	MaintMarginRatio float64 `json:"maintMarginRatio"`
	Cum              float64 `json:"cum"`
}

func (b *Client) SwapChangeInitialLeverage(symbol string, leverage int) (*ChangeLeverageResponse, error) {
	opts := ChnageLeverageOpts{
		Symbol:   symbol,
		Leverage: leverage,
	}
	res, err := b.do("future", http.MethodPost, "/fapi/v1/leverage", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &ChangeLeverageResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type ChnageLeverageOpts struct {
	Symbol   string `json:"json"`
	Leverage int    `json:"leverage"`
}

type ChangeLeverageResponse struct {
	Leverage         int    `json:"leverage"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	Symbol           string `json:"symbol"`
}
