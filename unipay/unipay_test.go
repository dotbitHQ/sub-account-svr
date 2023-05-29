package unipay

import (
	"das_sub_account/config"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/shopspring/decimal"
	"math/big"
	"testing"
)

func TestCreateOrder(t *testing.T) {
	config.Cfg.Server.UniPayUrl = "http://127.0.0.1:9090"
	resp, err := CreateOrder(ReqOrderCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "",
				Key:      "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891",
			},
		},
		BusinessId: BusinessIdAutoSubAccount,
		Amount:     decimal.NewFromInt(1e16),
		PayTokenId: "eth_eth",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(resp.OrderId, resp.PaymentAddress)
}

func TestRefundOrder(t *testing.T) {
	config.Cfg.Server.UniPayUrl = "http://127.0.0.1:9090"
	_, err := RefundOrder(ReqOrderRefund{
		BusinessId: BusinessIdAutoSubAccount,
		RefundList: []RefundInfo{{
			OrderId: "",
			PayHash: "",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOrderInfo(t *testing.T) {
	config.Cfg.Server.UniPayUrl = "http://127.0.0.1:9090"
	resp, err := GetPaymentInfo(ReqPaymentInfo{
		BusinessId:  BusinessIdAutoSubAccount,
		OrderIdList: []string{},
		PayHashList: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(resp.PaymentList)
}

func TestCalculatePrice(t *testing.T) {
	d := decimal.NewFromFloat(5126943111439236).Div(decimal.NewFromBigInt(big.NewInt(1), 18))
	t.Logf("%s", d.StringFixed(18))
}
