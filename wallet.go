package bnnapi

import (
	"net/http"
)

func (b *Client) Withdraw(asset, network, address, tag string, amount float64) (*WithdrawResponse, error) {
	opts := WithdrawOpts{
		Asset:   asset,
		Network: network,
		Address: address,
		Amount:  amount,
	}
	if tag != "" {
		opts.Tag = tag
	}
	res, err := b.do("spot", http.MethodPost, "sapi/v1/capital/withdraw/apply", opts, true, false) //margin
	if err != nil {
		return nil, err
	}
	resp := &WithdrawResponse{}
	err = json.Unmarshal(res, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type WithdrawOpts struct {
	Asset   string  `url:"coin"`
	Network string  `url:"network"`
	Address string  `url:"address"`
	Amount  float64 `url:"amount"`
	Tag     string  `url:"addressTag, omitempty"`
}

type WithdrawResponse struct {
	ID string `json:"id"`
}
