package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/scorpiotzh/toolib"
	"testing"
)

func TestAction(t *testing.T) {
	fmt.Println(common.Hex2Bytes(""))
}

func TestOwnerProfit(t *testing.T) {
	url := ApiUrl + "/owner/profit"
	req := handle.ReqOwnerProfit{ChainTypeAddress: core.ChainTypeAddress{
		Type: "blockchain",
		KeyInfo: core.KeyInfo{
			CoinType: common.CoinTypeEth,
			ChainId:  "",
			Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
		},
	}, Account: "00acc2022042902.bit"}
	var data handle.RespOwnerProfit
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestProfitWithdraw(t *testing.T) {
	var req handle.ReqProfitWithdraw
	req.IsWithdrawDotBit = true
	req.Account = "00acc2022042902.bit"
	req.ChainTypeAddress = core.ChainTypeAddress{
		Type: "blockchain",
		KeyInfo: core.KeyInfo{
			CoinType: "60",
			ChainId:  "5",
			Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
		},
	}

	url := ApiUrl + "/profit/withdraw"
	var data handle.RespProfitWithdraw
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestTransactionStatus(t *testing.T) {
	url := ApiUrl + "/transaction/status"
	req := handle.ReqTransactionStatus{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: common.CoinTypeEth,
				ChainId:  "",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Action:  common.DasActionCollectSubAccountProfit,
		Account: "00acc2022042902.bit",
	}

	var data handle.RespTransactionStatus

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(data))
}
