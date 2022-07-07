package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
	"testing"
)

func TestCustomScript(t *testing.T) {
	privateKey := ""

	args := common.Bytes2Hex(make([]byte, 33))
	//args = "0x01f15f519ecb226cd763b2bcbcab093e63f89100c07ac0caebc032c788b187ec99"
	fmt.Println(args)
	url := ApiUrl + "/custom/script/set"
	req := handle.ReqCustomScript{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: common.CoinTypeEth,
				ChainId:  "",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account:          "tzh2022070601.bit",
		CustomScriptArgs: args,
		CustomScriptConfig: map[uint8]witness.CustomScriptPrice{
			1: {5000000, 5000000},
			2: {4000000, 4000000},
			3: {3000000, 3000000},
			4: {2000000, 2000000},
			5: {1000000, 1000000},
		},
	}
	var data handle.RespCustomScript
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

func TestCustomScriptInfo(t *testing.T) {
	url := ApiUrl + "/custom/script/info"
	req := handle.ReqCustomScriptInfo{Account: "tzh2022070601.bit"}
	var data handle.RespCustomScriptInfo
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestSubAccountMintPrice(t *testing.T) {
	url := ApiUrl + "/sub/account/mint/price"
	req := handle.ReqSubAccountMintPrice{SubAccount: "01.tzh2022070601.bit"}
	var data handle.RespSubAccountMintPrice
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestConvertSubAccountCellOutputData(t *testing.T) {
	d := witness.ConvertSubAccountCellOutputData(common.Hex2Bytes("0x9c7d8e41528b34bae45e271e7fa38466c1a4dcc807d30a42093398edc593146d00a3e111000000000000000000000000"))
	fmt.Println(d.CustomScriptArgs, d.OwnerProfit, d.DasProfit, d.SmtRoot)
}
