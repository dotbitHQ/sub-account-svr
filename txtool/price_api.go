package txtool

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
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
	Price uint64
}

// PriceApiDefault
type PriceApiDefault struct{}

func (r *PriceApiDefault) GetPrice(p *ParamGetPrice) (*ResGetPrice, error) {
	var res ResGetPrice
	switch p.Action {
	case common.DasActionCreateSubAccount:
		if index := strings.Index(p.SubAccount, "."); index == -1 {
			return nil, fmt.Errorf("sub account invalid")
		} else {
			accLen := common.GetAccountLength(p.SubAccount[:index])
			switch accLen {
			case 1:
				res.Price = 16000000
			case 2:
				res.Price = 8000000
			case 3:
				res.Price = 4000000
			case 4:
				res.Price = 2000000
			default:
				res.Price = 1000000
			}
			log.Info("GetPrice:", res.Price, p.RegisterYears)
			res.Price *= p.RegisterYears
		}
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
	for _, v := range p.MintList {
		resPrice, err := p.PriceApi.GetPrice(&ParamGetPrice{
			Action:        p.Action,
			SubAccount:    v.Account,
			RegisterYears: v.RegisterYears,
		})
		if err != nil {
			return nil, fmt.Errorf("GetPrice err: %s", err.Error())
		}

		priceCkb := (resPrice.Price / p.Quote) * common.OneCkb
		log.Info("priceCkb:", priceCkb)
		dasCkb := (priceCkb / common.PercentRateBase) * uint64(p.NewSubAccountCustomPriceDasProfitRate)
		ownerCkb := priceCkb - dasCkb
		if dasCkb < common.OneCkb {
			return nil, fmt.Errorf("price is invalid: %s[%d<%d]", v.Account, dasCkb, common.OneCkb)
		}

		log.Info("price:", v.Account, v.RegisterYears, dasCkb, ownerCkb, dasCkb+ownerCkb)
		res.DasCapacity += dasCkb
		res.OwnerCapacity += ownerCkb
	}

	return &res, nil
}

type ParamCustomScriptMintTotalCapacity struct {
	Action                                common.DasAction
	PriceApi                              PriceApi
	MintList                              []tables.TableSmtRecordInfo
	Quote                                 uint64
	NewSubAccountCustomPriceDasProfitRate uint32
}
type ResCustomScriptMintTotalCapacity struct {
	DasCapacity   uint64
	OwnerCapacity uint64
}
