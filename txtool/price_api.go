package txtool

import (
	"context"
	"das_sub_account/dao"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
	"strings"
)

// PriceApi
type PriceApi interface {
	GetPrice(*ParamGetPrice) (*ResGetPrice, error)
}

type ParamGetPrice struct {
	Action        common.DasAction
	SubAccount    string
	RegisterYears uint64
}

type ResGetPrice struct {
	New              uint64
	Renew            uint64
	ActionTotalPrice uint64
}

// PriceApiDefault
type PriceApiDefault struct{}

func (r *PriceApiDefault) GetPrice(p *ParamGetPrice) (*ResGetPrice, error) {
	var res ResGetPrice

	index := strings.Index(p.SubAccount, ".")
	if index == -1 {
		return nil, fmt.Errorf("sub-account is invalid")
	}
	accLen := common.GetAccountLength(p.SubAccount[:index])

	// default price
	switch accLen {
	case 1:
		res.New, res.Renew = 16000000, 16000000
	case 2:
		res.New, res.Renew = 8000000, 8000000
	case 3:
		res.New, res.Renew = 4000000, 4000000
	case 4:
		res.New, res.Renew = 2000000, 2000000
	default:
		res.New, res.Renew = 1000000, 1000000
	}

	log.Info("PriceApiDefault:", p.Action, p.RegisterYears, res.New, res.Renew)

	switch p.Action {
	case common.DasActionCreateSubAccount:
		res.ActionTotalPrice = res.New * p.RegisterYears
	}

	return &res, nil
}

// PriceApiConfig
type PriceApiConfig struct {
	DasCore *core.DasCore
	DbDao   *dao.DbDao
}

func (r *PriceApiConfig) GetPrice(p *ParamGetPrice) (*ResGetPrice, error) {
	var res ResGetPrice

	index := strings.Index(p.SubAccount, ".")
	if index == -1 {
		return nil, fmt.Errorf("sub-account is invalid")
	}
	accLen := common.GetAccountLength(p.SubAccount[:index])

	// config price
	parentAccount := p.SubAccount[index+1:]
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(parentAccount))

	customScriptInfo, err := r.DbDao.GetCustomScriptInfo(parentAccountId)
	if err != nil {
		return nil, fmt.Errorf("GetCustomScriptInfo err: %s", err.Error())
	}
	outpoint := common.String2OutPointStruct(customScriptInfo.Outpoint)

	log.Info("PriceApiConfig:", customScriptInfo.Outpoint)
	resTx, err := r.DasCore.Client().GetTransaction(context.Background(), outpoint.TxHash)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction err: %s", err.Error())
	}

	_, customScriptConfig, err := witness.ConvertCustomScriptConfigByTx(resTx.Transaction)
	if err != nil {
		return nil, fmt.Errorf("ConvertCustomScriptConfigByTx err: %s", err.Error())
	}
	price, err := customScriptConfig.GetPrice(accLen)
	if err != nil {
		return nil, fmt.Errorf("price err: %s", err.Error())
	}
	res.New, res.Renew = price.New, price.Renew

	log.Info("PriceApiConfig:", p.Action, p.RegisterYears, res.New, res.Renew)
	switch p.Action {
	case common.DasActionCreateSubAccount:
		res.ActionTotalPrice = res.New * p.RegisterYears
	}

	return &res, nil
}

// GetCustomScriptMintTotalCapacity
func GetCustomScriptMintTotalCapacity(p *ParamCustomScriptMintTotalCapacity) (*ResCustomScriptMintTotalCapacity, error) {
	if p.PriceApi == nil {
		return nil, fmt.Errorf("PriceApi is nil")
	}
	if len(p.MintList) == 0 {
		return nil, fmt.Errorf("MintList is nil")
	}
	log.Info("GetCustomScriptMintTotalCapacity:", p.NewSubAccountCustomPriceDasProfitRate, p.Quote)

	var res ResCustomScriptMintTotalCapacity
	totalCKB := uint64(0)
	minDasCKb := uint64(0)
	for _, v := range p.MintList {
		resPrice, err := p.PriceApi.GetPrice(&ParamGetPrice{
			Action:        p.Action,
			SubAccount:    v.Account,
			RegisterYears: v.RegisterYears,
		})
		if err != nil {
			return nil, fmt.Errorf("GetPrice err: %s", err.Error())
		}

		priceCkb := (resPrice.ActionTotalPrice / p.Quote) * common.OneCkb
		log.Info("priceCkb:", priceCkb, p.Quote)
		totalCKB += priceCkb
		minDasCKb += v.RegisterYears * p.MinPriceCkb
	}
	res.DasCapacity = (totalCKB * uint64(p.NewSubAccountCustomPriceDasProfitRate)) / common.PercentRateBase
	res.OwnerCapacity = totalCKB - res.DasCapacity
	if res.DasCapacity < minDasCKb {
		res.DasCapacity = minDasCKb
		res.OwnerCapacity = totalCKB - res.DasCapacity
	}
	log.Info("price:", res.DasCapacity, res.OwnerCapacity)

	return &res, nil
}

type ParamCustomScriptMintTotalCapacity struct {
	Action                                common.DasAction
	PriceApi                              PriceApi
	MintList                              []tables.TableSmtRecordInfo
	Quote                                 uint64
	NewSubAccountCustomPriceDasProfitRate uint32
	MinPriceCkb                           uint64
}
type ResCustomScriptMintTotalCapacity struct {
	DasCapacity   uint64
	OwnerCapacity uint64
}
