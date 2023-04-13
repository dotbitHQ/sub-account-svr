package unipay

import (
	"das_sub_account/config"
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
	BusinessId string          `json:"business_id"`
	Amount     decimal.Decimal `json:"amount"`
	PayTokenId string          `json:"pay_token_id"`
}

type RespOrderCreate struct {
	OrderId        string `json:"order_id"`
	PaymentAddress string `json:"payment_address"`
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

type ReqOrderInfo struct {
	BusinessId string `json:"business_id"`
	OrderId    string `json:"order_id"`
}

type RespOrderInfo struct {
	BusinessId    string        `json:"business_id"`
	OrderId       string        `json:"order_id"`
	OrderStatus   OrderStatus   `json:"order_status"`
	PayStatus     PayStatus     `json:"pay_status"`
	PayHash       string        `json:"pay_hash"`
	PayHashStatus PayHashStatus `json:"pay_hash_status"`
	RefundStatus  RefundStatus  `json:"refund_status"`
	RefundHash    string        `json:"refund_hash"`
}

type PayStatus int

const (
	PayStatusUnpaid PayStatus = 0
	PayStatusPaid   PayStatus = 1
)

type OrderStatus int

const (
	OrderStatusNormal OrderStatus = 0
	OrderStatusCancel OrderStatus = 1
)

type PayHashStatus int

const (
	PayHashStatusPending PayHashStatus = 0
	PayHashStatusConfirm PayHashStatus = 1
	PayHashStatusFail    PayHashStatus = 2
)

type RefundStatus int

const (
	RefundStatusDefault    RefundStatus = 0
	RefundStatusUnRefunded RefundStatus = 1
	RefundStatusRefunded   RefundStatus = 2
)

func OrderInfo(req ReqOrderInfo) (resp RespOrderInfo, err error) {
	url := fmt.Sprintf("%s/v1/order/info", config.Cfg.Server.UniPayUrl)
	err = http_api.SendReq(url, &req, &resp)
	return
}
