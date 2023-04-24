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
		SubAccount:       "zh.sub-account-test.bit",
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

func TestAutoOrderHash(t *testing.T) {
	req := handle.ReqAutoOrderHash{
		ChainTypeAddress: ctaETH,
		OrderId:          "bd26fa24a1e25589b1cdbcbde49b097e",
		Hash:             "0x3cea874525d4bddd5a5afe5cac6f6058257340ead483ec063f1583789cab56d7",
	}
	data := handle.RespAutoOrderHash{}
	url := fmt.Sprintf("%s/auto/order/hash", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}
