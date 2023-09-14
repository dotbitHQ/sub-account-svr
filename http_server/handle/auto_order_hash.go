package handle

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"time"
)

type ReqAutoOrderHash struct {
	core.ChainTypeAddress
	OrderId string `json:"order_id"`
	Hash    string `json:"hash"`
}

type RespAutoOrderHash struct {
}

func (h *HttpHandle) AutoOrderHash(ctx *gin.Context) {
	var (
		funcName               = "AutoOrderHash"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqAutoOrderHash
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx)

	if err = h.doAutoOrderHash(&req, &apiResp); err != nil {
		log.Error("doAutoOrderHash err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoOrderHash(req *ReqAutoOrderHash, apiResp *api_code.ApiResp) error {
	var resp RespAutoOrderHash

	// check key info
	hexAddr, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("key-info[%s-%s] invalid", req.KeyInfo.CoinType, req.KeyInfo.Key))
		return nil
	}

	// check order
	orderInfo, err := h.DbDao.GetOrderByOrderID(req.OrderId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, fmt.Sprintf("Failed to search order: %s", req.OrderId))
		return fmt.Errorf("GetOrderByOrderID err: %s %s", err.Error(), req.OrderId)
	} else if orderInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, fmt.Sprintf("order[%s] does not exist", req.OrderId))
		return nil
	} else if !strings.EqualFold(orderInfo.PayAddress, hexAddr.AddressHex) {
		apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, fmt.Sprintf("invalid order[%s]", req.OrderId))
		return nil
	}

	// write back hash
	paymentInfo := tables.PaymentInfo{
		PayHash:       req.Hash,
		OrderId:       req.OrderId,
		PayHashStatus: tables.PayHashStatusPending,
		Timestamp:     time.Now().UnixMilli(),
	}
	if err := h.DbDao.CreatePaymentInfo(paymentInfo); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, fmt.Sprintf("Failed to write back hash: %s", req.OrderId))
		return fmt.Errorf("CreatePaymentInfo err: %s", err.Error())
	}
	apiResp.ApiRespOK(resp)
	return nil
}
