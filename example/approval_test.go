package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/scorpiotzh/toolib"
	"testing"
	"time"
)

func TestApprovalEnable(t *testing.T) {
	req := handle.ReqApprovalEnable{
		Platform: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				Key:      "0xe58673b9bF0a57398e0C8A1BDAe01EEB730177C8",
			},
		},
		Owner: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				Key:      "0xdeeFC10a42cD84c072f2b0e2fA99061a74A0698c",
			},
		},
		To: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				Key:      "0x52045950a5B582E9b426Ad89296c8970c96D09D9",
			},
		},
		Account:        "sub-account-test.bit",
		ProtectedUntil: uint64(time.Now().Add(time.Minute * 10).Unix()),
		SealedUntil:    uint64(time.Now().Add(time.Hour).Unix()),
		EvmChainId:     5,
	}
	data := handle.RespStatisticalInfo{}
	url := fmt.Sprintf("%s/approval/enable", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestApprovalDelay(t *testing.T) {
	req := handle.ReqApprovalDelay{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				Key:      "0xdeeFC10a42cD84c072f2b0e2fA99061a74A0698c",
			},
		},
		Account:     "sub-account-test.bit",
		SealedUntil: uint64(time.Now().Add(time.Hour).Unix()),
		EvmChainId:  5,
	}
	data := handle.RespStatisticalInfo{}
	url := fmt.Sprintf("%s/approval/delay", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestApprovalRevoke(t *testing.T) {
	req := handle.ReqApprovalRevoke{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				Key:      "0xe58673b9bF0a57398e0C8A1BDAe01EEB730177C8",
			},
		},
		Account: "sub-account-test.bit",
	}
	data := handle.RespStatisticalInfo{}
	url := fmt.Sprintf("%s/approval/delay", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestApprovalFulfill(t *testing.T) {
	req := handle.ReqApprovalFulfill{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				Key:      "0xdeeFC10a42cD84c072f2b0e2fA99061a74A0698c",
			},
		},
		Account: "sub-account-test.bit",
	}
	data := handle.RespStatisticalInfo{}
	url := fmt.Sprintf("%s/approval/fulfill", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}
