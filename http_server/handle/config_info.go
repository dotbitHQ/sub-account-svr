package handle

import (
	"das_sub_account/config"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
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
}

func (h *HttpHandle) ConfigInfo(ctx *gin.Context) {
	var (
		funcName               = "ConfigInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx)

	if err = h.doConfigInfo(&apiResp); err != nil {
		log.Error("doConfigInfo err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doConfigInfo(apiResp *api_code.ApiResp) error {
	var resp RespConfigInfo

	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}

	builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	resp.SubAccountBasicCapacity, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.BasicCapacity().RawData())
	resp.SubAccountPreparedFeeCapacity, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.PreparedFeeCapacity().RawData())
	resp.SubAccountCommonFee, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.CommonFee().RawData())
	resp.ManagementTimes = 10000

	mintPrice, _ := builder.NewSubAccountPrice()
	renewPrice, _ := builder.RenewSubAccountPrice()

	resp.MintCostsManually = decimal.NewFromInt(int64(mintPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
	resp.RenewCostsManually = decimal.NewFromInt(int64(renewPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)

	quoteCell, err := h.DasCore.GetQuoteCell()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return nil
	}
	quote := decimal.NewFromInt(int64(quoteCell.Quote()))
	resp.CkbQuote = quote.Div(decimal.NewFromInt(int64(common.OneCkb))).String()
	resp.SubAccountNewSubAccountPrice = config.PriceToCKB(mintPrice, quoteCell.Quote(), 1)
	resp.SubAccountRenewSubAccountPrice = config.PriceToCKB(renewPrice, quoteCell.Quote(), 1)

	resp.AutoMint.PaymentMinPrice = config.Cfg.Das.AutoMint.PaymentMinPrice
	resp.AutoMint.ServiceFeeRatio = fmt.Sprintf("%s%%", decimal.NewFromFloat(config.Cfg.Das.AutoMint.ServiceFeeRatio*100).String())
	apiResp.ApiRespOK(resp)
	return nil
}
