package txtool

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"math"
	"strings"
	"time"
)

type CsvRecord struct {
	Account      string
	AccountId    string
	TokenId      string
	Decimals     int32
	Address      string
	Amount       decimal.Decimal
	OriginAmount decimal.Decimal
	FeeRate      decimal.Decimal
	Fee          decimal.Decimal
	Ids          []uint64
}

func (s *SubAccountTxTool) StatisticsParentAccountPayment(parentAccount string, payment bool, endTime time.Time) (map[string]map[string]*CsvRecord, error) {
	var accountId string
	if parentAccount != "" {
		accountId = common.Bytes2Hex(common.GetAccountIdByAccount(parentAccount))
	}
	list, err := s.DbDao.FindOrderByPayment(endTime.UnixMilli(), accountId)
	if err != nil {
		return nil, err
	}

	tokens, err := s.DbDao.FindTokens()
	if err != nil {
		return nil, err
	}

	records := make(map[string]map[string]*CsvRecord)
	for _, v := range list {
		token, ok := tokens[v.TokenId]
		if !ok {
			err = fmt.Errorf("token_id: %s no exist", v.TokenId)
			return nil, err
		}

		var csvRecord *CsvRecord
		if _, ok = records[v.ParentAccountId]; !ok {
			records[v.ParentAccountId] = make(map[string]*CsvRecord)
		} else {
			csvRecord, ok = records[v.ParentAccountId][v.TokenId]
		}
		if !ok {
			accounts := strings.Split(v.Account, ".")
			account := accounts[len(accounts)-2] + "." + accounts[len(accounts)-1]
			csvRecord = &CsvRecord{}
			csvRecord.Account = account
			csvRecord.AccountId = v.ParentAccountId
			csvRecord.TokenId = v.TokenId
			csvRecord.Decimals = token.Decimals
			csvRecord.Ids = make([]uint64, 0)
			records[v.ParentAccountId][v.TokenId] = csvRecord
		}
		amount := decimal.Zero
		if v.CouponCode == "" {
			amount = v.Amount.Sub(v.PremiumAmount)
			couponMinPrice := decimal.NewFromFloat(1.09).Div(decimal.NewFromFloat(0.15))
			if v.USDAmount.GreaterThan(couponMinPrice) {
				amount = amount.Mul(decimal.NewFromFloat(0.85))
				csvRecord.FeeRate = decimal.NewFromFloat(0.15)
				csvRecord.Fee = amount.Mul(decimal.NewFromFloat(0.15))
			} else {
				amount = amount.Sub(decimal.NewFromFloat(1.09).Div(token.Price).Mul(decimal.NewFromInt(int64(token.Decimals))))
				csvRecord.Fee = decimal.NewFromFloat(1.09)
			}
		} else {
			couponSetInfo, err := s.DbDao.GetSetInfoByCoupon(v.CouponCode)
			if err != nil {
				return nil, err
			}
			if v.USDAmount.GreaterThan(couponSetInfo.Price) {
				amount = v.USDAmount.Sub(couponSetInfo.Price).Mul(decimal.NewFromFloat(0.85)).Div(token.Price).Mul(decimal.NewFromInt(int64(token.Decimals)))
				csvRecord.FeeRate = decimal.NewFromFloat(0.15)
				csvRecord.Fee = amount.Mul(decimal.NewFromFloat(0.15))
			}
		}
		csvRecord.Amount = csvRecord.Amount.Add(amount)
		csvRecord.Ids = append(csvRecord.Ids, v.Id)
	}

	recordsNew := make(map[string]map[string]*CsvRecord)
	for parentAccId, v := range records {
		for tokenId, record := range v {
			token, err := s.DbDao.GetTokenById(tables.TokenId(tokenId))
			if err != nil {
				return nil, err
			}

			price := record.Amount.Div(decimal.NewFromInt(int64(math.Pow10(int(record.Decimals))))).Mul(token.Price)
			if price.LessThan(decimal.NewFromInt(config.Cfg.Das.AutoMint.PaymentMinPrice)) {
				log.Warnf("account: %s, token_id: %s, amount: %s$ less than min price: %d$, skip it",
					record.Amount, record.TokenId, price, config.Cfg.Das.AutoMint.PaymentMinPrice)
				continue
			}

			recordKeys, ok := common.TokenId2RecordKeyMap[tokenId]
			if !ok {
				log.Warnf("token id: [%s] to record key mapping failed", tokenId)
				continue
			}
			recordInfo, err := s.DbDao.GetRecordsByAccountIdAndTypeAndLabel(record.AccountId, "address", common.LabelTopDID, recordKeys)
			if err != nil {
				log.Error(err)
				return nil, err
			}
			if recordInfo.Id == 0 {
				log.Warnf("account: %s, token_id: %s no address set, skip it", record.Account, tokenId)
				continue
			}
			record.Address = recordInfo.Value

			if _, ok := recordsNew[parentAccId]; !ok {
				recordsNew[parentAccId] = make(map[string]*CsvRecord)
			}
			if _, ok := recordsNew[parentAccId][tokenId]; !ok {
				recordsNew[parentAccId][tokenId] = &CsvRecord{}
			}
			recordsNew[parentAccId][tokenId] = record

			log.Infof("account: %s, token_id: %s, amount: %s, price: %s$", record.Account, tokenId, record.Amount, price)
		}
	}

	if !payment {
		return recordsNew, nil
	}

	err = s.DbDao.Transaction(func(tx *gorm.DB) error {
		for parentId, v := range recordsNew {
			for tokenId, record := range v {
				recordsNew[parentId][tokenId].Amount = record.Amount

				autoPaymentInfo := &tables.AutoPaymentInfo{
					Account:       record.Account,
					AccountId:     record.AccountId,
					TokenId:       record.TokenId,
					Amount:        record.Amount,
					OriginAmount:  record.OriginAmount,
					FeeRate:       record.FeeRate,
					Fee:           record.Fee,
					Address:       record.Address,
					PaymentDate:   time.Now(),
					PaymentStatus: tables.PaymentStatusSuccess,
				}
				if err := autoPaymentInfo.GenAutoPaymentId(); err != nil {
					return err
				}
				if err := tx.Create(autoPaymentInfo).Error; err != nil {
					return err
				}
				if len(record.Ids) > 0 {
					if err = tx.Model(&tables.OrderInfo{}).Where("id in (?)", record.Ids).
						Updates(map[string]interface{}{
							"auto_payment_id": autoPaymentInfo.AutoPaymentId,
						}).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return recordsNew, nil
}
