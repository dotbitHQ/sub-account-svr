package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"encoding/csv"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"math"
	"net/http"
	"strings"
	"time"
)

type ReqPaymentReportExport struct {
	Account string `json:"account"`
	End     string `json:"end" binding:"required"`
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

	end, err := time.Parse("2006-01-02", req.End)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	var accountId string
	if req.Account != "" {
		accountId = common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	}
	list, err := h.DbDao.FindOrderByPayment(end.Unix(), accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	records := make(map[string]*CsvRecord)
	for _, v := range list {
		token, err := h.DbDao.GetTokenById(tables.TokenId(v.TokenId))
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		recordKey := v.ParentAccountId + v.TokenId
		csvRecord, ok := records[recordKey]
		if !ok {
			accounts := strings.Split(v.Account, ".")
			account := accounts[len(accounts)-2] + "." + accounts[len(accounts)-1]
			csvRecord = &CsvRecord{}
			csvRecord.Account = account
			csvRecord.AccountId = v.ParentAccountId
			csvRecord.TokenId = v.TokenId
			csvRecord.Decimals = token.Decimals
			csvRecord.Ids = make([]uint64, 0)
			records[recordKey] = csvRecord
		}
		csvRecord.Amount = csvRecord.Amount.Add(v.Amount)
		csvRecord.Ids = append(csvRecord.Ids, v.Id)
	}

	recordsNew := make(map[string]*CsvRecord)
	for k, v := range records {
		token, err := h.DbDao.GetTokenById(tables.TokenId(v.TokenId))
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		price := v.Amount.Div(decimal.NewFromInt(int64(math.Pow10(int(v.Decimals))))).Mul(token.Price)
		if price.LessThan(decimal.NewFromInt(config.Cfg.Das.AutoMint.PaymentMinPrice)) {
			log.Warnf("account: %s, token_id: %s, amount: %s$ less than min price: %d$, skip it",
				v.Amount, v.TokenId, price, config.Cfg.Das.AutoMint.PaymentMinPrice)
			continue
		}

		recordKeys, ok := common.TokenId2RecordKeyMap[v.TokenId]
		if !ok {
			_ = ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("token id: [%s] to record key mapping failed", v.TokenId))
			return
		}
		record, err := h.DbDao.GetRecordsByAccountIdAndTypeAndLabel(v.AccountId, "address", LabelSubDIDApp, recordKeys)
		if err != nil {
			log.Error(err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if record.Id == 0 {
			log.Warnf("account: %s, token_id: %s no address set, skip it", v.Account, v.TokenId)
			continue
		}
		v.Address = record.Value
		recordsNew[k] = v

		log.Infof("account: %s, token_id: %s, amount: %s, price: %s$", v.Account, v.TokenId, v.Amount, price)
	}

	if config.Cfg.Das.AutoMint.ServiceFeeRatio < 0 || config.Cfg.Das.AutoMint.ServiceFeeRatio >= 1 {
		log.Errorf("service fee ratio: %f invalid", config.Cfg.Das.AutoMint.ServiceFeeRatio)
		_ = ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("service fee ratio: %f invalid", config.Cfg.Das.AutoMint.ServiceFeeRatio))
		return
	}

	err = h.DbDao.Transaction(func(tx *gorm.DB) error {
		for k, v := range recordsNew {
			amount := v.Amount.Mul(decimal.NewFromFloat(1 - config.Cfg.Das.AutoMint.ServiceFeeRatio))
			autoPaymentInfo := &tables.AutoPaymentInfo{
				Account:       v.Account,
				AccountId:     v.AccountId,
				TokenId:       v.TokenId,
				Amount:        amount,
				OriginAmount:  v.Amount,
				FeeRate:       decimal.NewFromFloat(config.Cfg.Das.AutoMint.ServiceFeeRatio),
				Address:       v.Address,
				PaymentDate:   time.Now(),
				PaymentStatus: tables.PaymentStatusSuccess,
			}
			if err := autoPaymentInfo.GenAutoPaymentId(); err != nil {
				return err
			}
			recordsNew[k].Amount = amount

			if err := tx.Create(autoPaymentInfo).Error; err != nil {
				return err
			}
			if err := h.DbDao.UpdateAutoPaymentIdById(v.Ids, autoPaymentInfo.AutoPaymentId); err != nil {
				return err
			}
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
