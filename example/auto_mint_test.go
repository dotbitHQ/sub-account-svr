package example

import (
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/scorpiotzh/toolib"
	"testing"
	"time"
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
		Account: "sub-account-test.bit",
		Page:    1,
		Size:    10,
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
		Title:            "title test1",
		Desc:             "desc test",
		Benefits:         "benefits test",
		Links: []tables.Link{{
			App:  "Twiter",
			Link: "https://twiter.com",
		}, {
			App:  "Telegram",
			Link: "https://telegram.com",
		}},
		BackgroundColor: "",
		Timestamp:       time.Now().UnixMilli(),
	}
	data := handle.RespMintConfigUpdate{}
	url := fmt.Sprintf("%s/mint/config/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
	//if err := doSign2(data.SignInfoList, private, false); err != nil {
	//	t.Fatal(err)
	//}
	//
	//if err := doTransactionSendNew(handle.ReqTransactionSend{
	//	SignInfoList: data.SignInfoList,
	//}); err != nil {
	//	t.Fatal(err)
	//}
}

func TestMintConfigGet(t *testing.T) {
	req := handle.ReqMintConfigGet{
		Account: "20230504.bit",
	}
	data := tables.MintConfig{}
	url := fmt.Sprintf("%s/mint/config/get", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestConfigAutoMintUpdate(t *testing.T) {
	req := handle.ReqConfigAutoMintUpdate{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
		Enable:           true,
	}
	data := handle.RespConfigAutoMintUpdate{}
	url := fmt.Sprintf("%s/config/auto_mint/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
	//if err := doSign2(data.SignInfoList, private, false); err != nil {
	//	t.Fatal(err)
	//}
	//
	//if err := doTransactionSendNew(handle.ReqTransactionSend{
	//	SignInfoList: data.SignInfoList,
	//}); err != nil {
	//	t.Fatal(err)
	//}
}

func TestConfigAutoMintGet(t *testing.T) {
	req := handle.ReqConfigAutoMintGet{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
	}
	data := handle.RespConfigAutoMintGet{}
	url := fmt.Sprintf("%s/config/auto_mint/get", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestCurrencyList(t *testing.T) {
	req := handle.ReqCurrencyList{
		//ChainTypeAddress: ctaETH,
		Account: "20230504.bit",
	}
	data := make([]tables.PaymentConfigElement, 0)
	url := fmt.Sprintf("%s/currency/list", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestCurrencyUpdate(t *testing.T) {
	req := handle.ReqCurrencyUpdate{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
		TokenId:          string(tables.TokenIdBnb),
		Enable:           true,
		Timestamp:        time.Now().UnixMilli(),
	}

	data := handle.RespCurrencyUpdate{}
	url := fmt.Sprintf("%s/currency/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
	if err := doSign2(data.SignInfoList, private, false); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSendNew(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPriceRuleList(t *testing.T) {
	req := handle.ReqPriceRuleList{
		Account: "20230504.bit",
	}
	data := handle.RespPriceRuleList{}
	url := fmt.Sprintf("%s/price/rule/list", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestPreservedRuleList(t *testing.T) {
	req := handle.ReqPreservedRuleList{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
	}
	data := handle.RespPriceRuleList{}
	url := fmt.Sprintf("%s/preserved/rule/list", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

var (
	private = ""
)

func TestPriceRuleUpdate(t *testing.T) {
	req := handle.ReqPriceRuleUpdate{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
		List: witness.SubAccountRuleSlice{{
			Index: 0,
			Name:  "test",
			Note:  "test",
			Price: 1e6,
			Ast: witness.AstExpression{
				Type:      witness.Operator,
				Name:      "",
				Symbol:    witness.Equ,
				Value:     nil,
				ValueType: "",
				Arguments: nil,
				Expressions: witness.AstExpressions{{
					Type:        witness.Variable,
					Name:        string(witness.AccountLength),
					Symbol:      "",
					Value:       nil,
					ValueType:   "",
					Arguments:   nil,
					Expressions: nil,
				}, {
					Type:        witness.Value,
					Name:        "",
					Symbol:      "",
					Value:       4,
					ValueType:   witness.Uint8,
					Arguments:   nil,
					Expressions: nil,
				}},
			},
			Status: 1,
		}},
	}
	//req.List = witness.SubAccountRuleSlice{}
	data := handle.RespConfigAutoMintUpdate{}
	url := fmt.Sprintf("%s/price/rule/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
	if err := doSign2(data.SignInfoList, private, false); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSendNew(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPreservedRuleUpdate(t *testing.T) {
	req := handle.ReqPriceRuleUpdate{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
		List: witness.SubAccountRuleSlice{{
			Index: 0,
			Name:  "test",
			Note:  "test",
			Price: 1e6,
			Ast: witness.AstExpression{
				Type:      witness.Operator,
				Name:      "",
				Symbol:    witness.Equ,
				Value:     nil,
				ValueType: "",
				Arguments: nil,
				Expressions: witness.AstExpressions{{
					Type:        witness.Variable,
					Name:        string(witness.AccountLength),
					Symbol:      "",
					Value:       nil,
					ValueType:   "",
					Arguments:   nil,
					Expressions: nil,
				}, {
					Type:        witness.Value,
					Name:        "",
					Symbol:      "",
					Value:       4,
					ValueType:   witness.Uint8,
					Arguments:   nil,
					Expressions: nil,
				}},
			},
			Status: 1,
		}},
	}
	data := handle.RespConfigAutoMintUpdate{}
	url := fmt.Sprintf("%s/preserved/rule/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
	if err := doSign2(data.SignInfoList, private, false); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSendNew(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestAutoPaymentList(t *testing.T) {
	req := handle.ReqAutoPaymentList{
		ChainTypeAddress: ctaETH,
		Account:          "20230504.bit",
		Page:             1,
		Size:             10,
	}
	data := handle.RespAutoPaymentList{}
	url := fmt.Sprintf("%s/auto/payment/list", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
}

func TestTime2(t *testing.T) {
	fmt.Println(tables.GetEfficientOrderTimestamp())
}
