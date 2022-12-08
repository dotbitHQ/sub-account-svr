package example

import (
	"das_sub_account/http_server/handle"
	"github.com/dotbitHQ/das-lib/core"
	"testing"
)

const (
	ApiUrlInternal = "http://127.0.0.1:9126/v1"
)

func TestInternalSubAccountMint(t *testing.T) {
	url := ApiUrlInternal + "/internal/sub/account/mint"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "20221130.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:       "test6.20221130.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
			},
			{
				Account:       "test7.20221130.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
			},
		},
	}

	var data handle.RespInternalSubAccountMintNew
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}
