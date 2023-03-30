package handle

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"math"
	"net/http"
	"reverse-svr/http_server/handle"
	"time"
)

type ReqPaymentReportExport struct {
	Account string `json:"account"`
	Begin   string `json:"begin" binding:"required"`
	End     string `json:"end" binding:"required"`
}

type CsvRecord struct {
	Account   string
	AccountId string
	TokenId   string
	Decimals  int
	Address   string
	Amount    float64
	Ids       []int64
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
	records := make(map[string]CsvRecord)
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
		csvRecord, ok := records[v.Account+v.TokenId]
		if !ok {
			csvRecord.Account = v.Account
			csvRecord.AccountId = v.AccountId
			csvRecord.TokenId = v.TokenId
			csvRecord.Address = record.Value
			csvRecord.Decimals = token.Decimals
			csvRecord.Ids = make([]int64, 0)
		}
		csvRecord.Amount += v.Amount
		csvRecord.Ids = append(csvRecord.Ids, v.Id)
	}

	err = h.DbDao.Transaction(func(tx *gorm.DB) error {
		for _, v := range records {
			autoPaymentInfo := tables.AutoPaymentInfo{
				Account:       v.Account,
				AccountId:     v.AccountId,
				TokenId:       v.TokenId,
				Amount:        v.Amount,
				Address:       v.Address,
				PaymentDate:   time.Now(),
				PaymentStatus: tables.PaymentStatusSuccess,
			}
			if err := autoPaymentInfo.GenAutoPaymentId(); err != nil {
				return err
			}
			if err := tx.Create(autoPaymentInfo).Error; err != nil {
				return err
			}
			if err := h.DbDao.UpdateAutoPaymentIdById(v.Ids, autoPaymentInfo.AutoPaymentId); err != nil {
				return err
			}
			return nil
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
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
	for _, v := range records {
		amount := fmt.Sprintf(fmt.Sprintf("%%.%df", v.Decimals), v.Amount/math.Pow10(v.Decimals))
		if err := w.Write([]string{v.Account, v.Address, v.TokenId, amount}); err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	w.Flush()
	ctx.Status(http.StatusOK)
}
