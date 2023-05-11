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
		SubAccount:       "001.20230504.bit",
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
		SubAccount:       "001.20230504.bit",
		TokenId:          tables.TokenIdBnb,
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
		OrderId:          "af7054eaf87de38a592bec32ff853fa6",
		Hash:             "0x0abd01adb4afbe65510e8d688c837358fd5403e79c4d86a9bb5604a01475b7d5",
	}
	data := handle.RespAutoOrderHash{}
	url := fmt.Sprintf("%s/auto/order/hash", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestOrderInfo(t *testing.T) {
	req := handle.ReqAutoOrderInfo{
		ChainTypeAddress: ctaETH,
		OrderId:          "af17fa3bc6b89ed8704c69a3c157d18b",
	}
	data := handle.RespAutoOrderInfo{}
	url := fmt.Sprintf("%s/auto/order/info", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}
