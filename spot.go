package bnnapi

import "net/http"

func (b *Client) SpotAccount() (*SpotAccountResponse, error) {
	res, err := b.do("spot", http.MethodGet, "api/v3/account", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := &SpotAccountResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type SpotAccountResponse struct {
	MakerCommission  int                   `json:"makerCommission"`
	TakerCommission  int                   `json:"takerCommission"`
	BuyerCommission  int                   `json:"buyerCommission"`
	SellerCommission int                   `json:"sellerCommission"`
	CanTrade         bool                  `json:"canTrade"`
	CanWithdraw      bool                  `json:"canWithdraw"`
	CanDeposit       bool                  `json:"canDeposit"`
	UpdateTime       int                   `json:"updateTime"`
	AccountType      string                `json:"accountType"`
	Balances         []SpotAccountBalances `json:"balances"`
	Permissions      []string              `json:"permissions"`
}

type SpotAccountBalances struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

type SubmitFlexibleLendingOpts struct {
	ProductID string  `json:"productId"`
	Amount    float64 `json:"amount"`
}

type SubmitFlexibleLendingResponse struct {
	Purchaseid int `json:"purchaseId"`
}

func (b *Client) SubmitFlexibleLending(productID string, amount float64) (*SubmitFlexibleLendingResponse, error) {
	opts := SubmitFlexibleLendingOpts{
		ProductID: productID,
		Amount:    amount,
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/lending/daily/purchase", opts, true, false)
	if err != nil {
		return nil, err
	}
	resp := &SubmitFlexibleLendingResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type RedeemFlexibleLendingOpts struct {
	ProductID string  `json:"productId"`
	Amount    float64 `json:"amount"`
	Type      string  `json:"type"`
}

// use "FAST" typ can redeem the asset right after, "NORMAL" will redeem at tomorrow 8am
func (b *Client) RedeenFlexibleLending(productID string, amount float64, typ string) error {
	opts := RedeemFlexibleLendingOpts{
		ProductID: productID,
		Amount:    amount,
		Type:      typ,
	}
	_, err := b.do("spot", http.MethodPost, "sapi/v1/lending/daily/redeem", opts, true, false)
	if err != nil {
		return err
	}
	return nil
}

type FlexibleLendingData struct {
	Asset                    string `json:"asset"`
	Avgannualinterestrate    string `json:"avgAnnualInterestRate"`
	Canpurchase              bool   `json:"canPurchase"`
	Canredeem                bool   `json:"canRedeem"`
	Dailyinterestperthousand string `json:"dailyInterestPerThousand"`
	Featured                 bool   `json:"featured"`
	Minpurchaseamount        string `json:"minPurchaseAmount"`
	Productid                string `json:"productId"`
	Purchasedamount          string `json:"purchasedAmount"`
	Status                   string `json:"status"`
	Uplimit                  string `json:"upLimit"`
	Uplimitperuser           string `json:"upLimitPerUser"`
}

func (b *Client) FlexibleLendingList() (*[]FlexibleLendingData, error) {
	res, err := b.do("spot", http.MethodGet, "sapi/v1/lending/daily/product/list", nil, true, false)
	if err != nil {
		return nil, err
	}
	resp := &[]FlexibleLendingData{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type OrderBookSnapShot struct {
	Lastupdateid float64    `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

func (b *Client) OrderBookSnapShot() (*OrderBookSnapShot, error) {
	res, err := b.do("spot", http.MethodGet, "api/v3/depth", nil, false, false)
	if err != nil {
		return nil, err
	}
	resp := &OrderBookSnapShot{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
