package example

import (
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/scorpiotzh/toolib"
	"testing"
)

var (
	ctaETH = core.ChainTypeAddress{
		Type: "blockchain",
		KeyInfo: core.KeyInfo{
			CoinType: common.CoinTypeEth,
			ChainId:  "",
			Key:      "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891",
		},
	}
)

func TestAutoAccountSearch(t *testing.T) {
	req := handle.ReqAutoAccountSearch{
		ChainTypeAddress: ctaETH,
		SubAccount:       "zh.sub-account-test.bit",
	}
	data := handle.RespAutoAccountSearch{}
	url := fmt.Sprintf("%s/auto/account/search", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestAutoOrderCreate(t *testing.T) {
	req := handle.ReqAutoOrderCreate{
		ChainTypeAddress: ctaETH,
		ActionType:       tables.ActionTypeMint,
		SubAccount:       "00007.0001.bit",
		TokenId:          tables.TokenIdEth,
		Years:            1,
	}
	data := handle.RespAutoOrderCreate{}
	url := fmt.Sprintf("%s/auto/order/create", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}
