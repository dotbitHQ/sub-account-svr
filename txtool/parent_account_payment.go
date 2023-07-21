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
		csvRecord.Amount = csvRecord.Amount.Add(v.Amount)
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

	if config.Cfg.Das.AutoMint.ServiceFeeRatio < 0 || config.Cfg.Das.AutoMint.ServiceFeeRatio >= 1 {
		log.Errorf("service fee ratio: %f invalid", config.Cfg.Das.AutoMint.ServiceFeeRatio)
		return nil, fmt.Errorf("service fee ratio: %f invalid", config.Cfg.Das.AutoMint.ServiceFeeRatio)
	}

	if payment {
		err = s.DbDao.Transaction(func(tx *gorm.DB) error {
			for parentId, v := range recordsNew {
				for tokenId, record := range v {
					amount := record.Amount.Mul(decimal.NewFromFloat(1 - config.Cfg.Das.AutoMint.ServiceFeeRatio))
					recordsNew[parentId][tokenId].Amount = amount

					autoPaymentInfo := &tables.AutoPaymentInfo{
						Account:       record.Account,
						AccountId:     record.AccountId,
						TokenId:       record.TokenId,
						Amount:        amount,
						OriginAmount:  record.Amount,
						FeeRate:       decimal.NewFromFloat(config.Cfg.Das.AutoMint.ServiceFeeRatio),
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
					if err := s.DbDao.UpdateAutoPaymentIdById(record.Ids, autoPaymentInfo.AutoPaymentId); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Error(err)
			return nil, err
		}
	} else {
		for parentId, v := range recordsNew {
			for tokenId, record := range v {
				amount := record.Amount.Mul(decimal.NewFromFloat(1 - config.Cfg.Das.AutoMint.ServiceFeeRatio))
				recordsNew[parentId][tokenId].Amount = amount
			}
		}
	}
	return recordsNew, nil
}
