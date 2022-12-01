package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"testing"
)

const (
	addr       = "0xc9f53b1d85356B60453F867610888D89a0B667Ad"
	privateKey = ""
)

func TestSubAccountEditNew(t *testing.T) {
	url := ApiUrl + "/new/sub/account/edit"
	var list = []string{
		"test1.20221130.bit",
		//"test2.20221130.bit",
		//"test3.20221130.bit",
		//"test4.20221130.bit",
	}
	for _, v := range list {
		req := handle.ReqSubAccountEdit{
			ChainTypeAddress: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      addr,
				},
			},
			Account: v,
			EditKey: common.EditKeyRecords,
			EditValue: handle.EditInfo{
				//Owner: core.ChainTypeAddress{
				//	Type: "blockchain",
				//	KeyInfo: core.KeyInfo{
				//		CoinType: "60",
				//		ChainId:  "5",
				//		Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
				//	},
				//},
				//Manager: core.ChainTypeAddress{
				//	Type: "blockchain",
				//	KeyInfo: core.KeyInfo{
				//		CoinType: "60",
				//		ChainId:  "5",
				//		Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
				//	},
				//},
				Records: []handle.EditRecord{
					{
						Key:   "twitter",
						Type:  "profile",
						Label: "",
						Value: "111",
						TTL:   "",
					},
				},
			},
		}

		var data handle.RespSubAccountEdit

		if err := doReq(url, req, &data); err != nil {
			t.Fatal(err)
		}

		if err := doSign(data.SignInfoList, privateKey); err != nil {
			t.Fatal(err)
		}

		if err := doTransactionSendNew(handle.ReqTransactionSend{
			SignInfoList: data.SignInfoList,
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSubAccountCreateNew(t *testing.T) {
	url := ApiUrl + "/new/sub/account/create"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      addr,
			},
		},
		Account:        "20221130.bit",
		SubAccountList: nil,
	}

	req.SubAccountList = make([]handle.CreateSubAccount, 0)
	for i := 1; i < 3; i++ {
		req.SubAccountList = append(req.SubAccountList, handle.CreateSubAccount{
			Account:       fmt.Sprintf("test02-%d.20221130.bit", i),
			RegisterYears: 1,
			ChainTypeAddress: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      addr,
				},
			},
		})
	}

	var data handle.RespSubAccountCreate

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	if err := doSign(data.SignInfoList, privateKey); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSendNew(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func doTransactionSendNew(req handle.ReqTransactionSend) error {
	url := ApiUrl + "/new/transaction/send"

	var data handle.RespTransactionSend

	if err := doReq(url, req, &data); err != nil {
		return fmt.Errorf("doReq err: %s", err.Error())
	}
	return nil
}
