package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
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
		Account:          "00acc2022042902.bit",
		CustomScriptArgs: args,
	}
	var data handle.RespCustomScript
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}
