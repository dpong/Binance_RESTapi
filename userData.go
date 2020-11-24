package bnnapi

import (
	"net/http"
)

func (b *Client) GetSwapListenKey() (*ListenKeyResponse, error) {
	res, err := b.do("future", http.MethodPost, "fapi/v1/listenKey", nil, false, true)
	if err != nil {
		return nil, err
	}
	resp := &ListenKeyResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) GetSpotListenKey() (*ListenKeyResponse, error) {
	res, err := b.do("spot", http.MethodPost, "api/v3/userDataStream", nil, false, true) //margin
	if err != nil {
		return nil, err
	}
	resp := &ListenKeyResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) GetMarginListenKey() (*ListenKeyResponse, error) {
	res, err := b.do("spot", http.MethodPost, "sapi/v1/userDataStream", nil, false, true) //margin
	if err != nil {
		return nil, err
	}
	resp := &ListenKeyResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Client) GetIsolatedMarginListenKey(symbol string) (*ListenKeyResponse, error) {
	opts := SwapMarkPriceOpts{
		Symbol: symbol,
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/userDataStream/isolated", opts, false, true) //margin
	if err != nil {
		return nil, err
	}
	resp := &ListenKeyResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type ListenKeyResponse struct {
	ListenKey string `json:"listenKey"`
}
