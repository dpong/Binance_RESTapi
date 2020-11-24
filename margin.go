package bnnapi

import (
	"errors"
	"fmt"
	"net/http"
)

func (b *Client) MarginTransfer(method, asset string, amount float64) (*TransferResponse, error) {
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
	res, err := b.do("spot", http.MethodPost, "sapi/v1/margin/transfer", opts, true, false)
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

func (b *Client) MarginIsolatedTransfer(method, symbol, asset string, amount float64) (*TransferResponse, error) {
	var from, to string
	if method == "in" {
		from = "SPOT"
		to = "ISOLATED_MARGIN"
	} else if method == "out" {
		to = "SPOT"
		from = "ISOLATED_MARGIN"
	} else {
		return nil, errors.New("can't recognize margin account transfer method")
	}
	opts := IsolatedTransferOpts{
		Asset:  asset,
		Symbol: symbol,
		Amount: amount,
		From:   from,
		To:     to,
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/margin/isolated/transfer", opts, true, false)
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

type IsolatedTransferOpts struct {
	Asset  string  `url:"asset"`
	Symbol string  `url:"symbol"`
	From   string  `url:"transFrom"`
	To     string  `url:"transTo"`
	Amount float64 `url:amount`
}

type TransferOpts struct {
	Asset        string  `url:"asset"`
	Amount       float64 `url:amount`
	TransferType int     `url:"type"` // 1: main -> cross, 2: cross -> main
}

type TransferResponse struct {
	ID int `json:"tranId"`
}

func (b *Client) MarginBorrow(isoSymbol, asset string, amount float64) (*TransferResponse, error) {
	opts := MarginBorrowOpts{
		Asset:  asset,
		Amount: amount,
	}
	if isoSymbol != "" {
		opts.IsolatedSymbol = isoSymbol
		opts.IsIsolated = "TRUE"
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/margin/loan", opts, true, false)
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

func (b *Client) MarginRepay(isoSymbol, asset string, amount float64) (*TransferResponse, error) {
	opts := MarginBorrowOpts{
		Asset:  asset,
		Amount: amount,
	}
	if isoSymbol != "" {
		opts.IsolatedSymbol = isoSymbol
		opts.IsIsolated = "TRUE"
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/margin/repay", opts, true, false)
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

type MarginBorrowOpts struct {
	Asset          string  `url:"asset"`
	Amount         float64 `url:"amount"`
	IsIsolated     string  `url:"isIsolated, omitempty"`
	IsolatedSymbol string  `url:"symbol, omitempty"`
}

func (b *Client) MarginAccount() (*MarginAccountResponse, error) {
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/account", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := &MarginAccountResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) MarginIsolatedAccount() (*MarginIsolatedAccountResponse, error) {
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/isolated/account", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := &MarginIsolatedAccountResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type MarginAccountResponse struct {
	BorrowEnabled       bool   `json:"borrowEnabled"`
	MarginLevel         string `json:"marginLevel"`
	TotalAssetOfBtc     string `json:"totalAssetOfBtc"`
	TotalLiabilityOfBtc string `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBtc  string `json:"totalNetAssetOfBtc"`
	TradeEnabled        bool   `json:"tradeEnabled"`
	TransferEnabled     bool   `json:"transferEnabled"`
	UserAssets          []struct {
		Asset    string `json:"asset"`
		Borrowed string `json:"borrowed"`
		Free     string `json:"free"`
		Interest string `json:"interest"`
		Locked   string `json:"locked"`
		NetAsset string `json:"netAsset"`
	} `json:"userAssets"`
}

type MarginIsolatedAccountResponse struct {
	Assets              []interface{} `json:"assets"`
	TotalAssetOfBtc     string        `json:"totalAssetOfBtc"`
	TotalLiabilityOfBtc string        `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBtc  string        `json:"totalNetAssetOfBtc"`
}

type IsoAccountTotalInfo struct {
	BaseAsset         IsoAccountAssetInfo `json:"baseAsset"`
	QuoteAsset        IsoAccountAssetInfo `json:"quoteAsset"`
	Symbol            string              `json:"symbol"`
	IsolatedCreated   bool                `json:"isolatedCreated"`
	MarginLevel       string              `json:"marginLevel"`
	MarginLevelStatus string              `json:"marginLevelStatus"`
	MarginRatio       string              `json:"marginRatio"`
	IndexPrice        string              `json:"indexPrice"`
	LiquidatePrice    string              `json:"liquidatePrice"`
	TradeEnabled      string              `json:"tradeEnabled"`
}

type IsoAccountAssetInfo struct {
	Asset         string `json:"asset"`
	BorrowEnabled bool   `json:"borrowEnabled"`
	Borrowed      string `json:"borrowed"`
	Free          string `json:"free"`
	Interest      string `json:"interest"`
	Locked        string `json:"locked"`
	NetAsset      string `json:"netAsset"`
	NetAssetOfBtc string `json:"netAssetOfBtc"`
	RepayEnabled  bool   `json:"repayEnabled"`
	TotalAsset    string `json:"totalAsset"`
}

func (b *Client) InterestHistory(isosymbol string) (*InterestHistoryResponse, error) {
	opts := InterestHistoryOpts{
		IsoSymbol: isosymbol,
		Size:      100,
	}
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/interestHistory", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &InterestHistoryResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type InterestHistoryOpts struct {
	IsoSymbol string `url:"isolatedSymbol"`
	Size      int    `url:"size"`
}

type InterestHistoryResponse struct {
	Rows []struct {
		IsolatedSymbol      string `json:"isolatedSymbol,omitempty"`
		Asset               string `json:"asset"`
		Interest            string `json:"interest"`
		InterestAccuredTime int64  `json:"interestAccuredTime"`
		InterestRate        string `json:"interestRate"`
		Principal           string `json:"principal"`
		Type                string `json:"type"`
	} `json:"rows"`
	Total int `json:"total"`
}

func (b *Client) MarginMaxBorrow(isoSymbol, asset string) (*MarginMaxResponse, error) {
	opts := MarginAssetStuffOpts{
		Asset: asset,
	}
	if isoSymbol != "" {
		opts.IsolatedSymbol = isoSymbol
	}
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/maxBorrowable", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &MarginMaxResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type MarginAssetStuffOpts struct {
	Asset          string `url:"asset"`
	IsolatedSymbol string `url:"isolatedSymbol, omitempty"`
}

type MarginMaxResponse struct {
	Amount string `json:"amount`
}

func (b *Client) MarginMaxTransferOut(isoSymbol, asset string) (*MarginMaxResponse, error) {
	opts := MarginAssetStuffOpts{
		Asset: asset,
	}
	if isoSymbol != "" {
		opts.IsolatedSymbol = isoSymbol
	}
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/maxTransferable", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &MarginMaxResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) MarginInterestRate() (*MarginInterestRateResponse, error) {
	res, err := b.do("special", http.MethodGet, "gateway-api/v1/public/isolated-margin/pair/vip-level", nil, false, false)
	if err != nil {
		return nil, err
	}
	resp := &MarginInterestRateResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type MarginInterestRateResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	MessageDetail string         `json:"messageDetail"`
	Data          []InterestData `json:"data"`
	Success       bool           `json:"success"`
}

type InterestData struct {
	Base        BaseDetail `json:"base"`
	Quote       BaseDetail `json:"quote"`
	MarginRatio string     `json:"marginRatio"`
}

type BaseDetail struct {
	Asset       string         `json:"assetName"`
	LevelDetail []InterestTier `json:"levelDetails"`
}

type InterestTier struct {
	Level             string `json:"level"`
	DailyInterestRate string `json:"interestRate"`
	MaxBorrowable     string `json:"maxBorrowable"`
}

type MarginAssetResponse struct {
	AssetFullName  string `json:"assetFullName"`
	AssetName      string `json:"assetName"`
	IsBorrowable   bool   `json:"isBorrowable"`
	IsMortgageable bool   `json:"isMortgageable"`
	UserMinBorrow  string `json:"userMinBorrow"`
	UserMinRepay   string `json:"userMinRepay"`
}

func (b *Client) MarginAllAsset() ([]*MarginAssetResponse, error) {
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/allAssets", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := []*MarginAssetResponse{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) IsolatedMarginAllSymbol() ([]*MarginTradePairResponse, error) {
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/isolated/allPairs", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := []*MarginTradePairResponse{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type MarginTradePairResponse struct {
	Base          string `json:"base"`
	ID            int64  `json:"id, omitempty"`
	IsBuyAllowed  bool   `json:"isBuyAllowed"`
	IsMarginTrade bool   `json:"isMarginTrade"`
	IsSellAllowed bool   `json:"isSellAllowed"`
	Quote         string `json:"quote"`
	Symbol        string `json:"symbol"`
}

func (b *Client) MarginAllTradePair() ([]*MarginTradePairResponse, error) {
	res, err := b.do("spot", http.MethodGet, "sapi/v1/margin/allPairs", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := []*MarginTradePairResponse{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type CreateIsolatedMarginResponse struct {
	Success bool   `json:"success"`
	Symbol  string `json:"symbol"`
}

type CreateIsolatedMarginOpts struct {
	Base  string `url:"base"`
	Quote string `url:"quote"`
}

func (b *Client) CreateIsolatedMargin(base, quote string) (*CreateIsolatedMarginResponse, error) {
	opts := CreateIsolatedMarginOpts{
		Base:  base,
		Quote: quote,
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/margin/isolated/create", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &CreateIsolatedMarginResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) IsolatedMarginTierData(symbol string) (*IsolatedMarginTierDataResponse, error) {
	path := fmt.Sprintf("gateway-api/v1/friendly/isolated-margin/ladder/%s", symbol)
	res, err := b.do("special", http.MethodGet, path, nil, false, false)
	if err != nil {
		return nil, err
	}
	resp := &IsolatedMarginTierDataResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type IsolatedMarginTierDataResponse struct {
	Code          string     `json:"code"`
	Message       string     `json:"message"`
	MessageDetail string     `json:"messageDetail"`
	Data          []TierData `json:"data"`
	Success       bool       `json:"success"`
}

type TierData struct {
	Ladder          string `json:"ladder"`
	Base            string `json:"base"`
	Quote           string `json:"quote"`
	BaseMaxBorrow   string `json:"baseMaxBorrow"`
	QuoteMaxBorrow  string `json:"quoteMaxBorrow"`
	InitialRatio    string `json:"normalBar"`
	MarginCallRatio string `json:"callBar"`
	PreLiqRatio     string `json:"preLiquidationBar"`
	ForceLiqRatio   string `json:"forceLiquidationBar"`
	MarginRatio     string `json:"marginRatio"`
}
