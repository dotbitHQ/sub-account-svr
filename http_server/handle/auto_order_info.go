package handle

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
)

type ReqAutoOrderInfo struct {
	core.ChainTypeAddress
	OrderId string `json:"order_id" binding:"required"`
}

type RespAutoOrderInfo struct {
	OrderId     string          `json:"order_id"`
	TokenId     string          `json:"token_id"`
	Amount      decimal.Decimal `json:"amount"`
	PayHash     string          `json:"pay_hash"`
	OrderStatus OrderStatus     `json:"order_status"`
}

type OrderStatus int

const (
	OrderStatusUnpaid            OrderStatus = 0
	OrderStatusConfirmingPayment OrderStatus = 1
	OrderStatusPaymentFail       OrderStatus = 2
	OrderStatusMinting           OrderStatus = 3
	OrderStatusMintFail          OrderStatus = 4
	OrderStatusMintOK            OrderStatus = 5
)

func (h *HttpHandle) AutoOrderInfo(ctx *gin.Context) {
	var (
		funcName               = "AutoOrderInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqAutoOrderInfo
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	//time.Sleep(time.Minute * 3)
	if err = h.doAutoOrderInfo(&req, &apiResp); err != nil {
		log.Error("doAutoOrderInfo err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoOrderInfo(req *ReqAutoOrderInfo, apiResp *api_code.ApiResp) error {
	var resp RespAutoOrderInfo
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
		smtRecord, err := h.DbDao.GetSmtRecordByOrderId(req.OrderId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query mint record")
			return fmt.Errorf("GetSmtRecordByOrderId err: %s", err.Error())
		}
		switch smtRecord.RecordType {
		case tables.RecordTypeDefault:
			resp.OrderStatus = OrderStatusMinting
		case tables.RecordTypeClosed:
			resp.OrderStatus = OrderStatusMintFail
		case tables.RecordTypeChain:
			resp.OrderStatus = OrderStatusMintOK
		}
	}

	if order.OrderStatus == tables.OrderStatusFail {
		resp.OrderStatus = OrderStatusMintFail
	}

	apiResp.ApiRespOK(resp)
	return nil
}
