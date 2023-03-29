package handle

import (
	"das_sub_account/http_server/api_code"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"math"
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
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", "attachment; filename=payments.csv")
	ctx.Header("Content-Type", "text/csv")

	w := csv.NewWriter(ctx.Writer)
	if err := w.Write([]string{"parent_account", "payment_address", "payment_type", "amount"}); err != nil {
		log.Error(err)
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	for _, v := range list {
		config, err := h.DbDao.GetUserPaymentConfig(v.AccountId)
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		cfg, ok := config.CfgMap[v.TokenId]
		if !ok {
			continue
		}
		if !cfg.Enable {
			continue
		}
		record, err := h.DbDao.GetRecordsByAccountIdAndTypeAndLabel(v.AccountId, "address", LabelSubDIDApp)
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if record.Id == 0 {
			continue
		}

		token, err := h.DbDao.GetTokenById(v.TokenId)
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if token.Id == 0 {
			err = fmt.Errorf("token_id: %s no exist", v.TokenId)
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		amount := fmt.Sprintf(fmt.Sprintf("%%.%df", token.Decimals), v.Amount/math.Pow10(token.Decimals))
		if err := w.Write([]string{v.Account, record.Value, v.TokenId, amount}); err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// TODO 更新打款記錄
	}
	w.Flush()
	ctx.Status(http.StatusOK)
}
