package handle

import (
	"context"
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"net/http"
)

type RespConfigInfo struct {
	SubAccountBasicCapacity        uint64 `json:"sub_account_basic_capacity"`
	SubAccountPreparedFeeCapacity  uint64 `json:"sub_account_prepared_fee_capacity"`
	SubAccountNewSubAccountPrice   uint64 `json:"sub_account_new_sub_account_price"`
	SubAccountRenewSubAccountPrice uint64 `json:"sub_account_renew_sub_account_price"`
	SubAccountCommonFee            uint64 `json:"sub_account_common_fee"`
	CkbQuote                       string `json:"ckb_quote"`
	AutoMint                       struct {
		PaymentMinPrice int64  `json:"payment_min_price"`
		ServiceFeeRatio string `json:"service_fee_ratio"`
	} `json:"auto_mint"`
	MintCostsManually  decimal.Decimal `json:"mint_costs_manually"`
	RenewCostsManually decimal.Decimal `json:"renew_costs_manually"`
	ManagementTimes    uint64          `json:"management_times"`
	Stripe             struct {
		PremiumPercentage decimal.Decimal `json:"premium_percentage"`
		PremiumBase       decimal.Decimal `json:"premium_base"`
	} `json:"stripe"`
	TokenList         []TokenData `json:"token_list"`
	MinChangeCapacity uint64      `json:"min_change_capacity"`
}

type TokenData struct {
	TokenId     tables.TokenId  `json:"token_id"`
	CoinType    common.CoinType `json:"coin_type"`
	Symbol      string          `json:"symbol"`
	Decimals    int32           `json:"decimals"`
	Price       decimal.Decimal `json:"price"`
	DisplayName string          `json:"display_name"`
	Icon        string          `json:"icon"`
}

func (h *HttpHandle) ConfigInfo(ctx *gin.Context) {
	var (
		funcName               = "ConfigInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx.Request.Context())

	if err = h.doConfigInfo(ctx.Request.Context(), &apiResp); err != nil {
		log.Error("doConfigInfo err:", err.Error(), funcName, clientIp, ctx.Request.Context())
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doConfigInfo(ctx context.Context, apiResp *api_code.ApiResp) error {
	var resp RespConfigInfo

	err := h.checkSystemUpgrade(apiResp)
	if err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}

	var builder *witness.ConfigCellDataBuilder

	errWg := &errgroup.Group{}
	errWg.Go(func() error {
		builder, err = h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
		if err != nil {
			return err
		}
		quoteCell, err := h.DasCore.GetQuoteCell()
		if err != nil {
			return err
		}
		quote := decimal.NewFromInt(int64(quoteCell.Quote()))

		mintPrice, _ := builder.NewSubAccountPrice()
		renewPrice, _ := builder.RenewSubAccountPrice()
		resp.CkbQuote = quote.Div(decimal.NewFromInt(int64(common.OneCkb))).String()
		resp.SubAccountNewSubAccountPrice = config.PriceToCKB(ctx, mintPrice, quoteCell.Quote(), 1)
		resp.SubAccountRenewSubAccountPrice = config.PriceToCKB(ctx, renewPrice, quoteCell.Quote(), 1)

		resp.MintCostsManually = decimal.NewFromInt(int64(mintPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
		resp.RenewCostsManually = decimal.NewFromInt(int64(renewPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
		return nil
	})

	errWg.Go(func() error {
		tokens, err := h.DbDao.GetTokenPriceList()
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "GetTokenPriceList err")
			return nil
		}
		for _, v := range tokens {
			resp.TokenList = append(resp.TokenList, TokenData{
				TokenId:     v.FormatTokenId(),
				CoinType:    v.CoinType,
				Symbol:      v.Symbol,
				Decimals:    v.Decimals,
				Price:       v.Price,
				DisplayName: v.DisplayName,
				Icon:        v.Icon,
			})
		}
		return nil
	})
	if err := errWg.Wait(); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "errWg.Wait err")
		return err
	}

	resp.SubAccountBasicCapacity, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.BasicCapacity().RawData())
	resp.SubAccountPreparedFeeCapacity, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.PreparedFeeCapacity().RawData())
	resp.SubAccountCommonFee, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.CommonFee().RawData())
	resp.ManagementTimes = 10000

	resp.AutoMint.PaymentMinPrice = config.Cfg.Das.AutoMint.PaymentMinPrice

	platformFeeRatio, err := decimal.NewFromString(config.Cfg.Das.AutoMint.PlatformFeeRatio)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "errWg.Wait err")
		return err
	}
	serviceFeeRate, err := decimal.NewFromString(config.Cfg.Das.AutoMint.ServiceFeeRatio)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "errWg.Wait err")
		return err
	}
	feeRatio := platformFeeRatio.Add(serviceFeeRate).Mul(decimal.NewFromInt(100)).String()
	resp.AutoMint.ServiceFeeRatio = fmt.Sprintf("%s%%", feeRatio)

	resp.Stripe.PremiumPercentage = config.Cfg.Stripe.PremiumPercentage
	resp.Stripe.PremiumBase = config.Cfg.Stripe.PremiumBase
	resp.MinChangeCapacity = common.DasLockWithBalanceTypeMinCkbCapacity

	apiResp.ApiRespOK(resp)
	return nil
}
