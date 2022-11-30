package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"testing"
)

func TestSubAccountCreateNew(t *testing.T) {
	url := ApiUrl + "/new/sub/account/create"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account:        "20221130.bit",
		SubAccountList: nil,
	}

	req.SubAccountList = make([]handle.CreateSubAccount, 0)
	for i := 0; i < 1; i++ {
		req.SubAccountList = append(req.SubAccountList, handle.CreateSubAccount{
			Account:       fmt.Sprintf("test01-%d.20221130.bit", i),
			RegisterYears: 1,
			ChainTypeAddress: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
				},
			},
		})
	}

	var data handle.RespSubAccountCreate

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	//if err := doSign(data.SignInfoList, ""); err != nil {
	//	t.Fatal(err)
	//}
	//
	//if err := doTransactionSend(handle.ReqTransactionSend{
	//	SignInfoList: data.SignInfoList,
	//}); err != nil {
	//	t.Fatal(err)
	//}
}
