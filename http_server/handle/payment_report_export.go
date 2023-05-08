package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"math"
	"net/http"
	"time"
)

type ReqPaymentReportExport struct {
	Account string `json:"account"`
	//Begin   string `json:"begin" binding:"required"`
	End string `json:"end" binding:"required"`
}

type CsvRecord struct {
	Account   string
	AccountId string
	TokenId   string
	Decimals  int32
	Address   string
	Amount    decimal.Decimal
	Ids       []uint64
}

func (h *HttpHandle) PaymentReportExport(ctx *gin.Context) {
	var (
		funcName               = "PaymentReportExport"
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

	//begin, err := time.Parse("2006-01-02", req.Begin)
	//if err != nil {
	//	apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
	//	ctx.JSON(http.StatusOK, apiResp)
	//	return
	//}
	end, err := time.Parse("2006-01-02", req.End)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	//req.Begin = begin.Format("2006-01-02 15:04:05")
	req.End = end.Format("2006-01-02 15:04:05")

	list, err := h.DbDao.FindOrderByPayment(req.End, req.Account)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	records := make(map[string]*CsvRecord)
	for _, v := range list {
		config, err := h.DbDao.GetUserPaymentConfig(v.AccountId)
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		_, ok := config.CfgMap[v.TokenId]
		if !ok {
			log.Infof("account: %s, token_id: %s no config set, skip it", v.Account, v.TokenId)
			continue
		}
		record, err := h.DbDao.GetRecordsByAccountIdAndTypeAndLabel(v.AccountId, "address", LabelSubDIDApp)
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if record.Id == 0 {
			log.Infof("account: %s, token_id: %s no address set, skip it", v.Account, v.TokenId)
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
			csvRecord = &CsvRecord{}
			csvRecord.Account = v.Account
			csvRecord.AccountId = v.AccountId
			csvRecord.TokenId = v.TokenId
			csvRecord.Address = record.Value
			csvRecord.Decimals = token.Decimals
			csvRecord.Ids = make([]uint64, 0)
			records[v.Account+v.TokenId] = csvRecord
		}
		csvRecord.Amount = csvRecord.Amount.Add(v.Amount)
		csvRecord.Ids = append(csvRecord.Ids, v.Id)
	}

	recordsNew := make(map[string]*CsvRecord)
	for k, v := range records {
		token, err := h.DbDao.GetTokenById(v.TokenId)
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if v.Amount.Div(decimal.NewFromInt(int64(v.Decimals))).Mul(token.Price).LessThan(decimal.NewFromInt(50)) {
			continue
		}
		recordsNew[k] = v
	}

	err = h.DbDao.Transaction(func(tx *gorm.DB) error {
		for k, v := range recordsNew {
			amount := v.Amount.Mul(decimal.NewFromFloat(1 - config.Cfg.Das.AutoMint.ServiceFeeRatio))
			recordsNew[k].Amount = amount
			autoPaymentInfo := &tables.AutoPaymentInfo{
				Account:       v.Account,
				AccountId:     v.AccountId,
				TokenId:       v.TokenId,
				Amount:        amount,
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
