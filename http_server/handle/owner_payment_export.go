package handle

import (
	"das_sub_account/http_server/api_code"
	"encoding/csv"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"math"
	"net/http"
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
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

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
		amount := v.Amount.DivRound(decimal.NewFromInt(int64(math.Pow10(int(v.Decimals)))), v.Decimals)
		if err := w.Write([]string{v.Account, v.Address, v.TokenId, amount.String()}); err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	w.Flush()
	ctx.Status(http.StatusOK)
}
