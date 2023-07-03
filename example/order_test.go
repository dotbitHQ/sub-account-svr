package example

import (
	"context"
	"das-pay/chain/chain_evm"
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/types"
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
		SubAccount:       "001.10086.bit",
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

func TestRule(t *testing.T) {
	client, err := getClientTestnet2()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := client.GetTransaction(context.Background(), types.HexToHash("0x454e3f1a4394979030a5b296b518ecb390a53da102127e3fb240bf850b8f6f82"))
	if err != nil {
		t.Fatal(err)
	}
	var rulePrice witness.SubAccountRuleEntity
	if err = rulePrice.ParseFromTx(tx.Transaction, common.ActionDataTypeSubAccountPreservedRules); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(&rulePrice.Rules))
	subAccount := "002.10988.bit"
	fmt.Println(common.Bytes2Hex(common.GetAccountIdByAccount(subAccount)))
	hit, index, err := rulePrice.Hit(subAccount)
	fmt.Println(hit, index)
}

func TestAutoOrderMint(t *testing.T) {
	req := handle.ReqAutoOrderCreate{
		ChainTypeAddress: ctaETH,
		ActionType:       tables.ActionTypeMint,
		SubAccount:       "t10.rt01.bit",
		TokenId:          tables.TokenIdEth,
		Years:            1,
	}
	data := handle.RespAutoOrderCreate{}
	url := fmt.Sprintf("%s/auto/order/create", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))

	ethClient, err := chain_evm.Initialize(context.Background(), "https://rpc.ankr.com/eth_goerli", 0)
	if err != nil {
		t.Fatal(err)
	}
	to := data.PaymentAddress
	nonce, err := ethClient.NonceAt(ctaETH.KeyInfo.Key)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := ethClient.NewTransaction(ctaETH.KeyInfo.Key, to, data.Amount, []byte(data.OrderId), nonce, 0)
	if err != nil {
		t.Fatal(err)
	}
	tx, err = ethClient.SignWithPrivateKey(private, tx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(tx.Hash().Hex())
	if err := ethClient.SendTransaction(tx); err != nil {
		fmt.Println("err:", err)
		t.Fatal(err)
	}

	hashReq := handle.ReqAutoOrderHash{
		ChainTypeAddress: ctaETH,
		OrderId:          data.OrderId,
		Hash:             tx.Hash().Hex(),
	}
	resp := handle.RespAutoOrderHash{}
	if err := http_api.SendReq(fmt.Sprintf("%s/auto/order/hash", ApiUrl), &hashReq, &resp); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&resp))
}

func TestAutoOrderRenew(t *testing.T) {
	req := handle.ReqAutoOrderCreate{
		ChainTypeAddress: ctaETH,
		ActionType:       tables.ActionTypeRenew,
		SubAccount:       "t12.rt01.bit",
		TokenId:          tables.TokenIdEth,
		Years:            1,
	}
	data := handle.RespAutoOrderCreate{}
	url := fmt.Sprintf("%s/auto/order/create", ApiUrl)
	if err := http_api.SendReq(url, &req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&data))

	ethClient, err := chain_evm.Initialize(context.Background(), "https://rpc.ankr.com/eth_goerli", 0)
	if err != nil {
		t.Fatal(err)
	}
	to := data.PaymentAddress
	nonce, err := ethClient.NonceAt(ctaETH.KeyInfo.Key)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := ethClient.NewTransaction(ctaETH.KeyInfo.Key, to, data.Amount, []byte(data.OrderId), nonce, 0)
	if err != nil {
		t.Fatal(err)
	}
	tx, err = ethClient.SignWithPrivateKey(private, tx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(tx.Hash().Hex())
	if err := ethClient.SendTransaction(tx); err != nil {
		fmt.Println("err:", err)
		t.Fatal(err)
	}

	hashReq := handle.ReqAutoOrderHash{
		ChainTypeAddress: ctaETH,
		OrderId:          data.OrderId,
		Hash:             tx.Hash().Hex(),
	}
	resp := handle.RespAutoOrderHash{}
	if err := http_api.SendReq(fmt.Sprintf("%s/auto/order/hash", ApiUrl), &hashReq, &resp); err != nil {
		t.Fatal(err)
	}
	fmt.Println("data:", toolib.JsonString(&resp))
}
