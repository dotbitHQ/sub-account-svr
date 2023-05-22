package example

import (
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/scorpiotzh/toolib"
	"strconv"
	"testing"
	"time"
)

func TestStatisticalInfo(t *testing.T) {
	req := handle.ReqStatisticalInfo{
		Account: "20230504.bit",
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
		Account: "20230504.bit",
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
		Account: "10086.bit",
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
		Account:          "sub-account-test.bit",
		List:             witness.SubAccountRuleSlice{},
	}
	for i := 0; i < 170; i++ {
		req.List = append(req.List, &witness.SubAccountRule{
			Index: uint32(i),
			Name:  fmt.Sprintf("test rule %d", i),
			Note: "A name that is not priced will not be automatically distributed; " +
				" name that meets more than one price rule may be automatically distributed at any one of the multiple prices it meets.",
			Price:  10,
			Status: 1,
			Ast: witness.AstExpression{
				Type:   witness.Operator,
				Symbol: witness.And,
				Expressions: witness.AstExpressions{
					{
						Type:   witness.Operator,
						Symbol: witness.Equ,
						Expressions: witness.AstExpressions{
							{
								Type: witness.Variable,
								Name: string(witness.AccountLength),
							},
							{
								Type:      witness.Value,
								ValueType: witness.Uint32,
								Value:     uint32(i + 1),
							},
						},
					},
					{
						Type: witness.Function,
						Name: string(witness.FunctionOnlyIncludeCharset),
						Arguments: []*witness.AstExpression{
							{
								Type: witness.Variable,
								Name: string(witness.AccountChars),
							},
							{
								Type:      witness.Value,
								ValueType: witness.Charset,
								Value:     common.AccountCharTypeEn,
							},
						},
					},
					{
						Type: witness.Function,
						Name: string(witness.FunctionIncludeWords),
						Arguments: []*witness.AstExpression{
							{
								Type: witness.Variable,
								Name: string(witness.Account),
							},
							{
								Type:      witness.Value,
								ValueType: witness.StringArray,
								Value:     []string{"test1", "test2", "test3"}},
						},
					},
				},
			},
		})
	}
	data := handle.RespConfigAutoMintUpdate{}
	url := fmt.Sprintf("%s/price/rule/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
	if err := doSign(data.SignInfoList, private); err != nil {
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
		Account: "20230504.bit",
		Page:    1,
		Size:    10,
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

func TestPriceRuleUpdateSizeLimit(t *testing.T) {
	req := handle.ReqPriceRuleUpdate{
		ChainTypeAddress: ctaETH,
		Account:          "sub-account-test.bit",
		List:             witness.SubAccountRuleSlice{},
	}

	num := 111

	for i := 0; i < num-1; i++ {
		req.List = append(req.List, &witness.SubAccountRule{
			Index: uint32(i),
			Name:  fmt.Sprintf("test rule %d", i),
			Note: "A name that is not priced will not be automatically distributed; " +
				" name that meets more than one price rule may be automatically distributed at any one of the multiple prices it meets.",
			Price:  0.0033,
			Status: 1,
			Ast: witness.AstExpression{
				Type:   witness.Operator,
				Symbol: witness.And,
				Expressions: witness.AstExpressions{
					{
						Type:   witness.Operator,
						Symbol: witness.Equ,
						Expressions: witness.AstExpressions{
							{
								Type: witness.Variable,
								Name: string(witness.AccountLength),
							},
							{
								Type:      witness.Value,
								ValueType: witness.Uint32,
								Value:     uint32(i + 1),
							},
						},
					},
					{
						Type: witness.Function,
						Name: string(witness.FunctionOnlyIncludeCharset),
						Arguments: []*witness.AstExpression{
							{
								Type: witness.Variable,
								Name: string(witness.AccountChars),
							},
							{
								Type:      witness.Value,
								ValueType: witness.Charset,
								Value:     common.AccountCharTypeEn,
							},
						},
					},
					{
						Type: witness.Function,
						Name: string(witness.FunctionIncludeWords),
						Arguments: []*witness.AstExpression{
							{
								Type: witness.Variable,
								Name: string(witness.Account),
							},
							{
								Type:      witness.Value,
								ValueType: witness.StringArray,
								Value:     []string{"test1", "test2", "test3"}},
						},
					},
				},
			},
		})
	}

	rule := &witness.SubAccountRule{
		Index: uint32(num - 1),
		Name:  "test in list rule",
		Note: "A name that is not priced will not be automatically distributed; " +
			" name that meets more than one price rule may be automatically distributed at any one of the multiple prices it meets.",
		Price:  0.0033,
		Status: 1,
		Ast: witness.AstExpression{
			Type: witness.Function,
			Name: string(witness.FunctionInList),
			Arguments: witness.AstExpressions{
				{
					Type: witness.Variable,
					Name: string(witness.Account),
				},
				{
					Type:      witness.Value,
					ValueType: witness.BinaryArray,
					Value:     []string{},
				},
			},
		},
	}
	for i := 0; i < 999; i++ {
		rule.Ast.Arguments[1].Value = append(rule.Ast.Arguments[1].Value.([]string), fmt.Sprintf("testprice%d", i))
	}
	req.List = append(req.List, rule)

	data := handle.RespConfigAutoMintUpdate{}
	url := fmt.Sprintf("%s/price/rule/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
	if err := doSign(data.SignInfoList, private); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSendNew(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPriceRuleUpdateInListLimit(t *testing.T) {
	req := handle.ReqPriceRuleUpdate{
		ChainTypeAddress: ctaETH,
		Account:          "sub-account-test.bit",
		List:             witness.SubAccountRuleSlice{},
	}

	rule := &witness.SubAccountRule{
		Index: 0,
		Name:  "test in list rule",
		Note: "A name that is not priced will not be automatically distributed; " +
			" name that meets more than one price rule may be automatically distributed at any one of the multiple prices it meets.",
		Price:  10,
		Status: 1,
		Ast: witness.AstExpression{
			Type: witness.Function,
			Name: string(witness.FunctionInList),
			Arguments: witness.AstExpressions{
				{
					Type: witness.Variable,
					Name: string(witness.Account),
				},
				{
					Type:      witness.Value,
					ValueType: witness.BinaryArray,
					Value:     []string{},
				},
			},
		},
	}
	for i := 0; i < 999; i++ {
		rule.Ast.Arguments[1].Value = append(rule.Ast.Arguments[1].Value.([]string), fmt.Sprintf("test%d", i))
	}
	req.List = append(req.List, rule)

	data := handle.RespConfigAutoMintUpdate{}
	url := fmt.Sprintf("%s/price/rule/update", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))
	if err := doSign(data.SignInfoList, private); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSendNew(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestMintAccount(t *testing.T) {
	records := make([]*tables.TableSmtRecordInfo, 0)
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount("sub-account-test.bit"))

	owner := core.DasAddressHex{
		DasAlgorithmId: 5,
		AddressHex:     "",
	}

	daf := core.DasAddressFormat{DasNetType: common.DasNetTypeTestnet2}
	args, err := daf.HexToArgs(owner, owner)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		account := "testprice" + strconv.Itoa(i) + ".sub-account-test.bit"
		accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
		accountCharset, err := common.GetAccountCharSetList(account)
		if err != nil {
			t.Fatal(err)
		}
		for idx := range accountCharset {
			if idx <= 8 {
				accountCharset[idx].CharSetName = common.AccountCharTypeEn
			} else {
				accountCharset[idx].CharSetName = common.AccountCharTypeDigit
			}
		}

		content, _ := json.Marshal(accountCharset)
		records = append(records, &tables.TableSmtRecordInfo{
			SvrName:         "svr1",
			AccountId:       accountId,
			RecordType:      tables.RecordTypeDefault,
			MintType:        tables.MintTypeAutoMint,
			OrderID:         "",
			Action:          common.DasActionUpdateSubAccount,
			ParentAccountId: parentAccountId,
			Account:         account,
			Content:         string(content),
			RegisterYears:   1,
			RegisterArgs:    common.Bytes2Hex(args),
			Timestamp:       time.Now().UnixNano() / 1e6,
			SubAction:       common.SubActionCreate,
		})
	}

	db, err := toolib.NewGormDB("", "", "", "", 100, 50)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(records).Error; err != nil {
		t.Fatal(err)
	}
}
