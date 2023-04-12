package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
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
	PayStatus    tables.PayStatus    `json:"pay_status"`
	PayHash      string              `json:"pay_hash"`
	RefundStatus tables.RefundStatus `json:"refund_status"`
	RefundHash   string              `json:"refund_hash"`
}

type ReqUniPayNotice struct {
	OrderId    string    `json:"order_id"`
	BusinessId string    `json:"business_id"`
	EventType  EventType `json:"event_type"`
	OrderInfo  OrderInfo `json:"order_info"`
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
	order, err := h.DbDao.GetOrderByOrderID(req.OrderId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to get order")
		return fmt.Errorf("GetOrderByOrderID err: %s", err.Error())
	} else if order.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, fmt.Sprintf("order[%s] not exist", req.OrderId))
		return nil
	}

	// check event type
	switch req.EventType {
	case EventTypeOrderPay:
		owner := core.DasAddressHex{
			DasAlgorithmId: order.AlgorithmId,
			AddressHex:     order.PayAddress,
		}
		args, err := h.DasCore.Daf().HexToArgs(owner, owner)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("Failed to get register args"))
			return fmt.Errorf("HexToArgs err: %s", err.Error())
		}
		charsetList, err := h.DasCore.GetAccountCharSetList(order.Account)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("Failed to get account charset list"))
			return fmt.Errorf("GetAccountCharSetList err: %s", err.Error())
		}
		content, err := json.Marshal(charsetList)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to json.Marshal charset list")
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
		if err := h.DbDao.UpdateOrderStatusOk(order.OrderId, smtRecord); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, fmt.Sprintf("Failed to update order status[%s]", order.OrderId))
			return fmt.Errorf("UpdateOrderStatusOk err: %s", err.Error())
		}
	case EventTypeOrderRefund:
		// todo
	default:
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("EventType[%s] invalid", req.EventType))
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}
