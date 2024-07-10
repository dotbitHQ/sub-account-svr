package handle

import (
	"context"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"das_sub_account/unipay"
	"fmt"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type EventType string

const (
	EventTypeOrderPay       EventType = "ORDER.PAY"
	EventTypeOrderRefund    EventType = "ORDER.REFUND"
	EventTypePaymentDispute EventType = "PAYMENT.DISPUTE"
)

type EventInfo struct {
	EventType    EventType           `json:"event_type"`
	OrderId      string              `json:"order_id"`
	PayStatus    tables.PayStatus    `json:"pay_status"`
	PayHash      string              `json:"pay_hash"`
	PayAddress   string              `json:"pay_address"`
	RefundStatus tables.RefundStatus `json:"refund_status"`
	RefundHash   string              `json:"refund_hash"`
}

type ReqUniPayNotice struct {
	BusinessId string      `json:"business_id"`
	EventList  []EventInfo `json:"event_list"`
}

type RespUniPayNotice struct {
}

func (h *HttpHandle) UniPayNotice(ctx *gin.Context) {
	var (
		funcName               = "UniPayNotice"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqUniPayNotice
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx.Request.Context())

	if err = h.doUniPayNotice(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doUniPayNotice err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doUniPayNotice(ctx context.Context, req *ReqUniPayNotice, apiResp *api_code.ApiResp) error {
	var resp RespUniPayNotice

	// check BusinessId
	if req.BusinessId != unipay.BusinessIdAutoSubAccount {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("BusinessId[%s] invalid", req.BusinessId))
		return nil
	}
	// check order id
	for _, v := range req.EventList {
		switch v.EventType {
		case EventTypeOrderPay:
			if err := unipay.DoPaymentConfirm(h.DasCore, h.DbDao, v.OrderId, v.PayHash); err != nil {
				log.Error(ctx, "DoPaymentConfirm err: ", err.Error())
				notify.SendLarkErrNotify("DoPaymentConfirm", err.Error())
			}
		case EventTypeOrderRefund:
			if err := h.DbDao.UpdateRefundStatusToRefunded(v.PayHash, v.OrderId, v.RefundHash); err != nil {
				log.Error(ctx, "UpdateRefundStatusToRefunded err: ", err.Error())
				notify.SendLarkErrNotify("UpdateRefundStatusToRefunded", err.Error())
			}
		case EventTypePaymentDispute:
			if err := h.DbDao.UpdatePayHashStatusToFailByDispute(v.PayHash, v.OrderId); err != nil {
				log.Error(ctx, "UpdatePayHashStatusToFailByDispute err: ", err.Error())
				notify.SendLarkErrNotify("UpdatePayHashStatusToFailByDispute", err.Error())
			}
		default:
			log.Error(ctx, "EventType invalid:", v.EventType)
		}
	}

	apiResp.ApiRespOK(resp)
	return nil
}
