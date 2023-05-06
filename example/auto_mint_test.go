package example

import (
	"das_sub_account/http_server/handle"
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
