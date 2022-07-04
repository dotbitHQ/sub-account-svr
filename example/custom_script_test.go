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
	args := common.Bytes2Hex(make([]byte, 33))
	args = "0x01f15f519ecb226cd763b2bcbcab093e63f89100c07ac0caebc032c788b187ec99"
	fmt.Println(args)
	url := ApiUrl + "/sub/account/custom/script"
	req := handle.ReqCustomScript{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: common.CoinTypeEth,
				ChainId:  "",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account:          "aaatzh0630.bit",
		CustomScriptArgs: args,
	}
	var data handle.RespCustomScript
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	if err := doSign(data.SignInfoList, ""); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSend(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestConvertSubAccountCellOutputData(t *testing.T) {
	d := witness.ConvertSubAccountCellOutputData(common.Hex2Bytes("0x9c7d8e41528b34bae45e271e7fa38466c1a4dcc807d30a42093398edc593146d00a3e111000000000000000000000000"))
	fmt.Println(d.CustomScriptArgs, d.OwnerProfit, d.DasProfit, d.SmtRoot)
}
