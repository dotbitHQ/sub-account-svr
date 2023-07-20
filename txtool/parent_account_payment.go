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
	Account   string
	AccountId string
	TokenId   string
	Decimals  int32
	Address   string
	Amount    decimal.Decimal
	Ids       []uint64
}

func (s *SubAccountTxTool) StatisticsParentAccountPayment(parentAccount string, payment bool, endTime time.Time) (map[string]*CsvRecord, error) {
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

	records := make(map[string]*CsvRecord)
	for _, v := range list {
		token, ok := tokens[v.TokenId]
		if !ok {
			err = fmt.Errorf("token_id: %s no exist", v.TokenId)
			return nil, err
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
		token, err := s.DbDao.GetTokenById(tables.TokenId(v.TokenId))
		if err != nil {
			return nil, err
		}

		price := v.Amount.Div(decimal.NewFromInt(int64(math.Pow10(int(v.Decimals))))).Mul(token.Price)
		if price.LessThan(decimal.NewFromInt(config.Cfg.Das.AutoMint.PaymentMinPrice)) {
			log.Warnf("account: %s, token_id: %s, amount: %s$ less than min price: %d$, skip it",
				v.Amount, v.TokenId, price, config.Cfg.Das.AutoMint.PaymentMinPrice)
			continue
		}

		recordKeys, ok := common.TokenId2RecordKeyMap[v.TokenId]
		if !ok {
			return nil, fmt.Errorf("token id: [%s] to record key mapping failed", v.TokenId)
		}
		record, err := s.DbDao.GetRecordsByAccountIdAndTypeAndLabel(v.AccountId, "address", common.LabelTopDID, recordKeys)
		if err != nil {
			log.Error(err)
			return nil, err
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
		return nil, fmt.Errorf("service fee ratio: %f invalid", config.Cfg.Das.AutoMint.ServiceFeeRatio)
	}

	if payment {
		err = s.DbDao.Transaction(func(tx *gorm.DB) error {
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
				if err := s.DbDao.UpdateAutoPaymentIdById(v.Ids, autoPaymentInfo.AutoPaymentId); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Error(err)
			return nil, err
		}
	} else {
		for k, v := range recordsNew {
			amount := v.Amount.Mul(decimal.NewFromFloat(1 - config.Cfg.Das.AutoMint.ServiceFeeRatio))
			recordsNew[k].Amount = amount
		}
	}
	return recordsNew, nil
}
