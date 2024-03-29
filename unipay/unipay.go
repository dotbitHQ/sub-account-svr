package unipay

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/shopspring/decimal"
)

const (
	BusinessIdAutoSubAccount = "auto-sub-account"
)

type ReqOrderCreate struct {
	core.ChainTypeAddress
	BusinessId        string            `json:"business_id"`
	Amount            decimal.Decimal   `json:"amount"`
	PayTokenId        tables.TokenId    `json:"pay_token_id"`
	PaymentAddress    string            `json:"payment_address"`
	PremiumPercentage decimal.Decimal   `json:"premium_percentage"`
	PremiumBase       decimal.Decimal   `json:"premium_base"`
	PremiumAmount     decimal.Decimal   `json:"premium_amount"`
	MetaData          map[string]string `json:"meta_data"`
}

type RespOrderCreate struct {
	OrderId               string `json:"order_id"`
	PaymentAddress        string `json:"payment_address"`
	ContractAddress       string `json:"contract_address"`
	StripePaymentIntentId string `json:"stripe_payment_intent_id"`
	ClientSecret          string `json:"client_secret"`
}

func CreateOrder(req ReqOrderCreate) (resp RespOrderCreate, err error) {
	url := fmt.Sprintf("%s/v1/order/create", config.Cfg.Server.UniPayUrl)
	err = http_api.SendReq(url, &req, &resp)
	return
}

type RefundInfo struct {
	OrderId string `json:"order_id"`
	PayHash string `json:"pay_hash"`
}

type ReqOrderRefund struct {
	BusinessId string       `json:"business_id"`
	RefundList []RefundInfo `json:"refund_list"`
}

type RespOrderRefund struct {
}

func RefundOrder(req ReqOrderRefund) (resp RespOrderRefund, err error) {
	url := fmt.Sprintf("%s/v1/order/refund", config.Cfg.Server.UniPayUrl)
	err = http_api.SendReq(url, &req, &resp)
	return
}

type ReqPaymentInfo struct {
	BusinessId  string   `json:"business_id"`
	OrderIdList []string `json:"order_id_list"`
	PayHashList []string `json:"pay_hash_list"`
}

type RespPaymentInfo struct {
	PaymentList []PaymentInfo `json:"payment_list"`
}

type PaymentInfo struct {
	OrderId       string               `json:"order_id"`
	PayHash       string               `json:"pay_hash"`
	PayAddress    string               `json:"pay_address"`
	PayHashStatus tables.PayHashStatus `json:"pay_hash_status"`
	RefundStatus  tables.RefundStatus  `json:"refund_status"`
	RefundHash    string               `json:"refund_hash"`
}

func GetPaymentInfo(req ReqPaymentInfo) (resp RespPaymentInfo, err error) {
	url := fmt.Sprintf("%s/v1/payment/info", config.Cfg.Server.UniPayUrl)
	err = http_api.SendReq(url, &req, &resp)
	return
}

type ReqOrderInfo struct {
	BusinessId string `json:"business_id"`
	OrderId    string `json:"order_id"`
}

type RespOrderInfo struct {
	OrderId         string `json:"order_id"`
	PaymentAddress  string `json:"payment_address"`
	ContractAddress string `json:"contract_address"`
	ClientSecret    string `json:"client_secret"`
}

func GetOrderInfo(req ReqOrderInfo) (resp RespOrderInfo, err error) {
	url := fmt.Sprintf("%s/v1/order/info", config.Cfg.Server.UniPayUrl)
	err = http_api.SendReq(url, &req, &resp)
	return
}
