package example

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/parnurzeal/gorequest"
	"github.com/scorpiotzh/toolib"
	"testing"
	"time"
)

const (
	ApiUrl = "https://test-subaccount-api.did.id/v1"
)

func doReq(url string, req, data interface{}) error {
	var resp api_code.ApiResp
	resp.Data = &data

	_, body, errs := gorequest.New().Post(url).Timeout(time.Minute * 20).SendStruct(&req).EndStruct(&resp)
	if errs != nil {
		return fmt.Errorf("%v , %s", errs, string(body))
	}
	fmt.Println("=========== doReq:", toolib.JsonString(data))
	if resp.ErrNo != api_code.ApiCodeSuccess {
		return fmt.Errorf("%d - %s", resp.ErrNo, resp.ErrMsg)
	}

	return nil
}

func TestVersion(t *testing.T) {
	url := ApiUrl + "/version"
	var req = handle.ReqVersion{}
	var data handle.RespVersion

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestSmtInfo(t *testing.T) {
	url := ApiUrl + "/smt/info"
	var req = handle.ReqSmtInfo{}
	var data handle.RespSmtInfo

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

}

func TestConfigInfo(t *testing.T) {
	url := ApiUrl + "/config/info"
	var data handle.RespConfigInfo

	if err := doReq(url, nil, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(data))
}

func TestAccountList(t *testing.T) {
	url := ApiUrl + "/account/list"
	req := handle.ReqAccountList{
		Pagination: handle.Pagination{Page: 1, Size: 5},
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
	}

	var data handle.RespAccountList

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(data))
}

func TestAccountDetail(t *testing.T) {
	url := ApiUrl + "/account/detail"
	req := handle.ReqAccountDetail{
		Account: "tzh2022070601.bit",
	}

	var data handle.RespAccountDetail

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(data))
}

func TestSubAccountList(t *testing.T) {
	url := ApiUrl + "/sub/account/list"
	req := handle.ReqSubAccountList{
		Pagination: handle.Pagination{Page: 1, Size: 10},
		Account:    "0001.bit",
	}

	var data handle.RespSubAccountList

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(data))
}
