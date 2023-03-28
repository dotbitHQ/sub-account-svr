package handle

import (
	"das_sub_account/http_server/api_code"
	"github.com/gin-gonic/gin"
	"net/http"
	"reverse-svr/http_server/handle"
)

type ReqPaymentReportExport struct {
	Account string `json:"account"`
	Begin   string `json:"begin" binding:"required"`
	End     string `json:"end" binding:"required"`
}

func (h *HttpHandle) PaymentReportExport(ctx *gin.Context) {
	var (
		funcName = "PaymentReportExport"
		clientIp = handle.GetClientIp(ctx)
		req      ReqPaymentReportExport
		apiResp  api_code.ApiResp
		err      error
	)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	list, err := h.DbDao.FindOrderPaymentInfo(req.Begin, req.End, req.Account)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	apiResp.ApiRespOK(list)
	ctx.JSON(http.StatusOK, apiResp)
}
