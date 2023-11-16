package handle

import (
	"das_sub_account/tables"
	"das_sub_account/unipay"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
)

type ReqCouponOrderInfo struct {
	core.ChainTypeAddress
	OrderId string `json:"order_id" binding:"required"`
	Account string `json:"account" binding:"required"`
}

type RespCouponOrderInfo struct {
	OrderId         string          `json:"order_id"`
	TokenId         string          `json:"token_id"`
	PaymentAddress  string          `json:"payment_address"`
	ContractAddress string          `json:"contract_address"`
	ClientSecret    string          `json:"client_secret"`
	Amount          decimal.Decimal `json:"amount"`
	PayHash         string          `json:"pay_hash"`
	OrderStatus     OrderStatus     `json:"order_status"`
	Cid             string          `json:"cid"`
}

func (h *HttpHandle) CouponOrderInfo(ctx *gin.Context) {
	var (
		funcName               = "CouponOrderInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponOrderInfo
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req), ctx)

	if err = h.doCouponOrderInfo(&req, &apiResp); err != nil {
		log.Error("doAutoOrderInfo err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponOrderInfo(req *ReqCouponOrderInfo, apiResp *api_code.ApiResp) error {
	order, err := h.DbDao.GetOrderByOrderID(req.OrderId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order")
		return fmt.Errorf("GetOrderByOrderID err: %s", err.Error())
	}
	if order.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, "order not exist")
		return nil
	}

	// get order info
	orderInfo, err := unipay.GetOrderInfo(unipay.ReqOrderInfo{
		BusinessId: unipay.BusinessIdAutoSubAccount,
		OrderId:    req.OrderId,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return err
	}

	var resp RespCouponOrderInfo
	resp.OrderId = req.OrderId
	resp.TokenId = order.TokenId
	resp.Amount = order.Amount
	resp.PaymentAddress = orderInfo.PaymentAddress
	resp.ContractAddress = orderInfo.ContractAddress
	resp.ClientSecret = orderInfo.ClientSecret

	if order.TokenId == string(tables.TokenIdDp) {
		token, err := h.DbDao.GetTokenById(tables.TokenIdDp)
		if err != nil {
			return err
		}
		resp.Amount = resp.Amount.Div(decimal.New(1, token.Decimals))
	}

	// get payment
	paymentInfo, err := h.DbDao.GetPaymentInfoByOrderId(req.OrderId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query payment")
		return fmt.Errorf("GetPaymentInfoByOrderId err: %s", err.Error())
	}

	if paymentInfo.Id == 0 {
		resp.OrderStatus = OrderStatusUnpaid
	} else {
		resp.PayHash = paymentInfo.PayHash
		switch paymentInfo.PayHashStatus {
		case tables.PayHashStatusPending:
			resp.OrderStatus = OrderStatusConfirmingPayment
		case tables.PayHashStatusConfirmed:
			resp.OrderStatus = OrderStatusMinting
		case tables.PayHashStatusRejected:
			resp.OrderStatus = OrderStatusPaymentFail
		}
	}

	if resp.OrderStatus == OrderStatusMinting {
		setInfo, err := h.DbDao.GetCouponSetInfoByOrderId(req.OrderId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon set info")
			return fmt.Errorf("GetCouponSetInfoByOrderId err: %s", err.Error())
		}
		if setInfo.Id > 0 {
			resp.OrderStatus = OrderStatusMintOK
			resp.Cid = setInfo.Cid
		}
	}

	if order.OrderStatus == tables.OrderStatusFail {
		resp.OrderStatus = OrderStatusMintFail
	}
	apiResp.ApiRespOK(resp)
	return nil
}
