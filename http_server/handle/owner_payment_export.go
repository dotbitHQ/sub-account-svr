package handle

import (
	"encoding/csv"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"net/http"
	"strings"
	"time"
)

type ReqPaymentReportExport struct {
	Account string `json:"account"`
	End     string `json:"end" binding:"required"`
	Payment bool   `json:"payment"`
}

func (h *HttpHandle) OwnerPaymentExport(ctx *gin.Context) {
	var (
		funcName               = "OwnerPaymentExport"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqPaymentReportExport
		apiResp                api_code.ApiResp
		err                    error
	)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	req.Account = strings.ToLower(req.Account)

	end, err := time.Parse("2006-01-02", req.End)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	recordsNew, err := h.TxTool.StatisticsParentAccountPayment(req.Account, req.Payment, end)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
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
	for _, v := range recordsNew {
		for _, record := range v {
			amount := record.Amount.DivRound(decimal.New(1, record.Decimals), record.Decimals)
			if err := w.Write([]string{record.Account, record.Address, record.TokenId, amount.String()}); err != nil {
				log.Error(err)
				_ = ctx.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}
	}
	w.Flush()
	ctx.Status(http.StatusOK)
}
