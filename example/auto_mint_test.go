package example

import (
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/scorpiotzh/toolib"
	"testing"
)

func TestStatisticalInfo(t *testing.T) {
	req := handle.ReqStatisticalInfo{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
	}
	data := handle.RespStatisticalInfo{}
	url := fmt.Sprintf("%s/statistical/info", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestDistributionList(t *testing.T) {
	req := handle.ReqDistributionList{
		ChainTypeAddress: ctaETH,
		Account:          "sub-account-test.bit",
		Page:             1,
		Size:             10,
	}
	data := handle.RespDistributionList{}
	url := fmt.Sprintf("%s/distribution/list", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestMintConfigUpdate(t *testing.T) {
	req := handle.ReqMintConfigUpdate{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
		Title:            "title test",
		Desc:             "desc test",
		Benefits:         "benefits test",
		Links: []tables.Link{{
			App:  "Twiter",
			Link: "https://twiter.com",
		}},
	}
	data := ""
	url := fmt.Sprintf("%s/mint/config/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestConfigAutoMintUpdate(t *testing.T) {
	req := handle.ReqConfigAutoMintUpdate{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
		Enable:           false,
	}
	data := handle.RespConfigAutoMintUpdate{}
	url := fmt.Sprintf("%s/config/auto_mint/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}
