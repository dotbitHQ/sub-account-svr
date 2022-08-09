package example

import (
	"das_sub_account/http_server/handle"
	"github.com/dotbitHQ/das-lib/core"
	"testing"
)

func TestMintForAccountCheck(t *testing.T) {
	url := ApiUrl + "/sub/account/check"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "tzh20220809.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:          "test02.tzh20220809.bit",
				MintForAccount:   "test01.tzh20220809.bit",
				AccountCharStr:   nil,
				RegisterYears:    1,
				ChainTypeAddress: core.ChainTypeAddress{},
			},
			{
				Account:          "มิ์02ญิ.tzh20220809.bit",
				MintForAccount:   "",
				AccountCharStr:   nil,
				RegisterYears:    1,
				ChainTypeAddress: core.ChainTypeAddress{},
			},
		},
	}
	var data handle.RespSubAccountCheck

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestMintForAccount(t *testing.T) {
	privateKey := ""
	url := ApiUrl + "/sub/account/create"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "tzh20220809.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:          "test02.tzh20220809.bit",
				MintForAccount:   "test01.tzh20220809.bit",
				AccountCharStr:   nil,
				RegisterYears:    1,
				ChainTypeAddress: core.ChainTypeAddress{},
			},
			{
				Account:          "มิ์02ญิ.tzh20220809.bit",
				MintForAccount:   "",
				AccountCharStr:   nil,
				RegisterYears:    1,
				ChainTypeAddress: core.ChainTypeAddress{},
			},
		},
	}

	var data handle.RespSubAccountCreate

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	if err := doSign(data.SignInfoList, privateKey); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSend(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}

}
