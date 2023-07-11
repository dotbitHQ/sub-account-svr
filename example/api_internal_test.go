package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/scorpiotzh/toolib"
	"testing"
)

const (
	ApiUrlInternal = "http://127.0.0.1:8127/v1"
)

func TestInternalRecycleAccount(t *testing.T) {
	req := handle.ReqRecycleAccount{SubAccountIds: []string{}}
	fmt.Printf("curl -X POST %s/internal/recycle/account -d '%s'\n", ApiUrlInternal, toolib.JsonString(&req))
}

func TestInternalSubAccountMint(t *testing.T) {
	//url := ApiUrlInternal + "/internal/sub/account/mint"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891",
			},
		},
		Account: "10086.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:       "test1.10086.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891",
					},
				},
			},
			{
				Account:       "test2.10086.bit",
				RegisterYears: 2,
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
	fmt.Printf("curl -X POST http://127.0.0.1:9126/v1/internal/sub/account/mint -d '%s'\n", toolib.JsonString(&req))

	//var data handle.RespInternalSubAccountMintNew
	//if err := doReq(url, req, &data); err != nil {
	//	t.Fatal(err)
	//}
}
