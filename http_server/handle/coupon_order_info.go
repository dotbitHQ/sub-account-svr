package handle

import (
	"das_sub_account/tables"
	"errors"
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
}

type RespCouponOrderInfo struct {
	OrderId     string          `json:"order_id"`
	TokenId     string          `json:"token_id"`
	Amount      decimal.Decimal `json:"amount"`
	PayHash     string          `json:"pay_hash"`
	OrderStatus OrderStatus     `json:"order_status"`
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
	var resp RespCouponOrderInfo
	// check key info
	_, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("key-info[%s-%s] invalid", req.KeyInfo.CoinType, req.KeyInfo.Key))
		return nil
	}

	// get order
	order, err := h.DbDao.GetOrderByOrderID(req.OrderId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order")
		return fmt.Errorf("GetOrderByOrderID err: %s", err.Error())
	}

	if order.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, "order not exist")
		return nil
	}
	resp.OrderId = req.OrderId
	resp.TokenId = order.TokenId
	resp.Amount = order.Amount

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
			if order.MetaData.Cid != setInfo.Cid {
				err := errors.New("order cid not match coupon set info cid")
				apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
				return err
			}
			resp.OrderStatus = OrderStatusMintOK
		}
	}

	if order.OrderStatus == tables.OrderStatusFail {
		resp.OrderStatus = OrderStatusMintFail
	}

	apiResp.ApiRespOK(resp)
	return nil
}
