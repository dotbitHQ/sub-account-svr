package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"das_sub_account/unipay"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

type EventType string

const (
	EventTypeOrderPay    EventType = "ORDER.PAY"
	EventTypeOrderRefund EventType = "ORDER.REFUND"
)

type OrderInfo struct {
	OrderId      string              `json:"order_id"`
	PayStatus    tables.PayStatus    `json:"pay_status"`
	PayHash      string              `json:"pay_hash"`
	RefundStatus tables.RefundStatus `json:"refund_status"`
	RefundHash   string              `json:"refund_hash"`
}

type ReqUniPayNotice struct {
	BusinessId string      `json:"business_id"`
	EventType  EventType   `json:"event_type"`
	Orders     []OrderInfo `json:"orders"`
}

type RespUniPayNotice struct {
}

func (h *HttpHandle) UniPayNotice(ctx *gin.Context) {
	var (
		funcName = "UniPayNotice"
		clientIp = GetClientIp(ctx)
		req      ReqUniPayNotice
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doUniPayNotice(&req, &apiResp); err != nil {
		log.Error("doUniPayNotice err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doUniPayNotice(req *ReqUniPayNotice, apiResp *api_code.ApiResp) error {
	var resp RespUniPayNotice

	// check BusinessId
	if req.BusinessId != unipay.BusinessIdAutoSubAccount {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("BusinessId[%s] invalid", req.BusinessId))
		return nil
	}
	// check order id
	switch req.EventType {
	case EventTypeOrderPay:
		for i := range req.Orders {
			if err := h.doPayConfirm(req.Orders[i]); err != nil {
				log.Error("doPayConfirm err: %s", err.Error())
				notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doPayConfirm", err.Error())
			}
		}
	case EventTypeOrderRefund:
		for _, v := range req.Orders {
			if err := h.DbDao.UpdateRefundStatusToRefunded(v.PayHash, v.OrderId); err != nil {
				log.Error("UpdateRefundStatusToRefunded err: %s", err.Error())
				notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "UpdateRefundStatusToRefunded", err.Error())
			}
		}
	default:
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("EventType[%s] invalid", req.EventType))
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doPayConfirm(reqOrder OrderInfo) error {
	order, err := h.DbDao.GetOrderByOrderID(reqOrder.OrderId)
	if err != nil {
		return fmt.Errorf("GetOrderByOrderID err: %s", err.Error())
	} else if order.Id == 0 {
		return fmt.Errorf("order[%s] not exist", reqOrder.OrderId)
	}

	paymentInfo := tables.PaymentInfo{
		PayHash:       reqOrder.PayHash,
		OrderId:       reqOrder.OrderId,
		PayHashStatus: tables.PayHashStatusConfirmed,
		Timestamp:     time.Now().Unix(),
	}

	owner := core.DasAddressHex{
		DasAlgorithmId: order.AlgorithmId,
		AddressHex:     order.PayAddress,
	}
	args, err := h.DasCore.Daf().HexToArgs(owner, owner)
	if err != nil {
		return fmt.Errorf("HexToArgs err: %s", err.Error())
	}
	charsetList, err := h.DasCore.GetAccountCharSetList(order.Account)
	if err != nil {
		return fmt.Errorf("GetAccountCharSetList err: %s", err.Error())
	}
	content, err := json.Marshal(charsetList)
	if err != nil {
		return fmt.Errorf("json Marshal err: %s", err.Error())
	}

	smtRecord := tables.TableSmtRecordInfo{
		SvrName:         config.Cfg.Slb.SvrName,
		AccountId:       order.AccountId,
		RecordType:      tables.RecordTypeDefault,
		MintType:        tables.MintTypeAutoMint,
		OrderID:         order.OrderId,
		Action:          common.DasActionUpdateSubAccount,
		ParentAccountId: order.GetParentAccountId(),
		Account:         order.Account,
		Content:         string(content),
		RegisterYears:   order.Years,
		RegisterArgs:    common.Bytes2Hex(args),
		Timestamp:       time.Now().UnixNano() / 1e6,
		SubAction:       common.SubActionCreate,
	}

	rowsAffected, sri, err := h.DbDao.UpdateOrderStatusOkWithSmtRecord(paymentInfo, smtRecord)
	if err != nil {
		return fmt.Errorf("UpdateOrderStatusOkWithSmtRecord err: %s", err.Error())
	} else if rowsAffected > 0 && sri.Id == 0 {
		log.Warnf("doUniPayNotice:", reqOrder.OrderId, rowsAffected)
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "multiple orders success", reqOrder.OrderId)
		// multiple orders from the same account are successful
	}
	return nil
}
