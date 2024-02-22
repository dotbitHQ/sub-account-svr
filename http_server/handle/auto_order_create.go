package handle

import (
	"crypto/md5"
	"das_sub_account/config"
	"das_sub_account/tables"
	"das_sub_account/unipay"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"strings"
	"time"
)

const (
	PaymentStatusNormal = 0
	PaymentWithout      = 1
)

type ReqAutoOrderCreate struct {
	core.ChainTypeAddress
	ActionType tables.ActionType `json:"action_type"`
	SubAccount string            `json:"sub_account" binding:"required"`
	TokenId    tables.TokenId    `json:"token_id"`
	Years      uint64            `json:"years" binding:"gt=0"`
	CouponCode string            `json:"coupon_code"`
}

type RespAutoOrderCreate struct {
	OrderId         string          `json:"order_id"`
	PaymentAddress  string          `json:"payment_address"`
	ContractAddress string          `json:"contract_address"`
	ClientSecret    string          `json:"client_secret"`
	Amount          decimal.Decimal `json:"amount"`
	PaymentStatus   int             `json:"payment_status"`
}

func (h *HttpHandle) AutoOrderCreate(ctx *gin.Context) {
	var (
		funcName               = "AutoOrderCreate"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqAutoOrderCreate
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx)

	if err = h.doAutoOrderCreate(&req, &apiResp); err != nil {
		log.Error("doAutoOrderCreate err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoOrderCreate(req *ReqAutoOrderCreate, apiResp *api_code.ApiResp) error {
	now := time.Now()
	// check key info
	hexAddr, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("key-info[%s-%s] invalid", req.KeyInfo.CoinType, req.KeyInfo.Key))
		return nil
	}
	// check sub_account name
	parentAccountId := h.checkSubAccountName(apiResp, req.SubAccount)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check parent account
	parentAccount, err := h.checkParentAccount(apiResp, parentAccountId)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check sub_account
	subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccount))
	accStatus, _, _, _, err := h.checkSubAccount(req.ActionType, apiResp, hexAddr, subAccountId)
	if err != nil {
		return err
	}
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	switch accStatus {
	case AccStatusMinting:
		apiResp.ApiRespErr(api_code.ApiCodeSubAccountMinting, fmt.Sprintf("sub-account[%s] is minting", req.SubAccount))
		return nil
	case AccStatusRenewing:
		apiResp.ApiRespErr(api_code.ApiCodeSubAccountRenewing, fmt.Sprintf("sub-account[%s] is renewing", req.SubAccount))
		return nil
	case AccStatusMinted:
		apiResp.ApiRespErr(api_code.ApiCodeSubAccountMinted, fmt.Sprintf("sub-account[%s] has been minted", req.SubAccount))
		return nil
	}

	// check switch
	autoDistribution, err := h.checkSwitch(parentAccountId, req.ActionType, apiResp)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// get max years
	if maxYear := h.getMaxYears(parentAccount); req.Years > maxYear {
		apiResp.ApiRespErr(api_code.ApiCodeBeyondMaxYears, "The main account is valid for less than one year")
		return nil
	}

	// check min price 0.99$
	builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to get config info")
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	// get rule price
	usdAmount, defaultRenewRule, err := h.getRulePrice(parentAccount.Account, parentAccountId, req.SubAccount, apiResp, req.ActionType, builder, autoDistribution)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	log.Info("usdAmount:", usdAmount.String(), req.Years)
	// total usd price
	usdAmount = usdAmount.Mul(decimal.NewFromInt(int64(req.Years)))

	// deduct coupons
	actualUsdPrice := usdAmount
	var couponInfo tables.CouponInfo
	if req.CouponCode != "" {
		lockKey := fmt.Sprintf("%x", md5.Sum([]byte("coupon:use:"+req.CouponCode)))
		if err := h.RC.Lock(lockKey); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to get lock")
			return fmt.Errorf("RC.Lock err: %s", err.Error())
		}
		defer func() {
			if err := h.RC.UnLock(lockKey); err != nil {
				log.Error("RC.UnLock err:", err.Error())
			}
		}()

		couponInfo, err = h.DbDao.GetCouponByCode(req.CouponCode)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
			return fmt.Errorf("GetCouponSetInfoByCode err: %s", err.Error())
		}
		if couponInfo.Id == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "coupon code not exist")
			return nil
		}
		if couponInfo.Status != tables.CouponStatusNormal {
			switch couponInfo.Status {
			case tables.CouponStatusUsed:
				apiResp.ApiRespErr(api_code.ApiCodeError500, "coupon code has been used")
			default:
				apiResp.ApiRespErr(api_code.ApiCodeError500, "coupon code status is not normal")
			}
			return nil
		}

		setInfo, err := h.DbDao.GetCouponSetInfo(couponInfo.Cid)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
			return fmt.Errorf("GetCouponSetInfo err: %s", err.Error())
		}
		if setInfo.OrderId == "" || setInfo.Status != tables.CouponSetInfoStatusSuccess {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "this coupon code can not use, because it order not paid")
			return nil
		}
		if setInfo.BeginAt > 0 && now.Before(time.UnixMilli(setInfo.BeginAt)) {
			apiResp.ApiRespErr(api_code.ApiCodeCouponOpenTimeNotArrived, "this coupon code can not use, because it not open")
			return nil
		}
		if now.After(time.UnixMilli(setInfo.ExpiredAt)) {
			apiResp.ApiRespErr(api_code.ApiCodeCouponExpired, "this coupon code can not use, because it expired")
			return nil
		}
		if setInfo.AccountId != parentAccountId {
			apiResp.ApiRespErr(api_code.ApiCodeCouponErrAccount, "this coupon code can not use, because it not belong to this account")
			return nil
		}

		if setInfo.Price.LessThan(usdAmount) {
			actualUsdPrice = usdAmount.Sub(setInfo.Price)
		} else {
			actualUsdPrice = decimal.Zero
		}
	}

	amount := decimal.Zero
	premiumPercentage := decimal.Zero
	premiumBase := decimal.Zero
	premiumAmount := decimal.Zero
	var tokenPrice tables.TTokenPriceInfo

	// usd price -> token price
	if actualUsdPrice.GreaterThan(decimal.Zero) {
		// check user payment config
		if defaultRenewRule {
			find := false
			for _, v := range config.Cfg.Das.AutoMint.SupportPaymentToken {
				if v == string(req.TokenId) {
					find = true
					break
				}
			}
			if !find {
				apiResp.ApiRespErr(api_code.ApiCodeNoSupportPaymentToken, "payment method not supported")
				return nil
			}
		} else {
			userConfig, err := h.DbDao.GetUserPaymentConfig(parentAccountId)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to search payment config")
				return fmt.Errorf("GetUserPaymentConfig err: %s", err.Error())
			} else if cfg, ok := userConfig.CfgMap[string(req.TokenId)]; !ok || !cfg.Enable {
				apiResp.ApiRespErr(api_code.ApiCodeTokenIdNotSupported, "payment method not supported")
				return nil
			}
		}

		newSubAccountPrice, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.NewSubAccountPrice().RawData())
		minPrice := decimal.NewFromInt(int64(newSubAccountPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
		if req.ActionType == tables.ActionTypeRenew {
			renewSubAccountPrice, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.RenewSubAccountPrice().RawData())
			minPrice = decimal.NewFromInt(int64(renewSubAccountPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
		}
		if minPrice.GreaterThan(usdAmount) {
			apiResp.ApiRespErr(api_code.ApiCodePriceRulePriceNotBeLessThanMin, "Pricing below minimum")
			return fmt.Errorf("price not be less than min: %s$", minPrice.String())
		}

		// usd price -> token price
		tokenPrice, err = h.DbDao.GetTokenById(req.TokenId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to search token price")
			return fmt.Errorf("GetTokenById err: %s", err.Error())
		} else if tokenPrice.Id == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeTokenIdNotSupported, "payment method not supported")
			return nil
		}
		amount = actualUsdPrice.Mul(decimal.New(1, tokenPrice.Decimals)).Div(tokenPrice.Price).Ceil()
		amount = RoundAmount(amount, req.TokenId)

		if req.TokenId == tables.TokenIdStripeUSD {
			premiumPercentage = config.Cfg.Stripe.PremiumPercentage
			premiumBase = config.Cfg.Stripe.PremiumBase
			premiumAmount = amount
			amount = amount.Mul(premiumPercentage.Add(decimal.NewFromInt(1))).Add(premiumBase.Mul(decimal.NewFromInt(100)))
			amount = decimal.NewFromInt(amount.Ceil().IntPart())
			premiumAmount = amount.Sub(premiumAmount)
			usdAmount = usdAmount.Mul(premiumPercentage.Add(decimal.NewFromInt(1))).Add(premiumBase.Mul(decimal.NewFromInt(100)))
		}
	}

	action := "mint"
	switch req.ActionType {
	case tables.ActionTypeRenew:
		action = "renew"
	}
	res, err := unipay.CreateOrder(unipay.ReqOrderCreate{
		ChainTypeAddress:  req.ChainTypeAddress,
		BusinessId:        unipay.BusinessIdAutoSubAccount,
		Amount:            amount,
		PayTokenId:        req.TokenId,
		PaymentAddress:    config.GetUnipayAddress(req.TokenId),
		PremiumPercentage: premiumPercentage,
		PremiumBase:       premiumBase,
		PremiumAmount:     premiumAmount,
		MetaData: map[string]string{
			"account":      req.SubAccount,
			"algorithm_id": hexAddr.ChainType.ToString(),
			"address":      req.ChainTypeAddress.KeyInfo.Key,
			"action":       action,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "600004") {
			apiResp.ApiRespErr(api_code.ApiCodePaymentMethodDisable, "This payment method is unavailable")
			return fmt.Errorf("unipay.CreateOrder err: %s", err.Error())
		}
		if strings.Contains(err.Error(), "600003") {
			apiResp.ApiRespErr(api_code.ApiCodeAmountIsTooLow, "Amount must not be lower than 0.52$")
			return fmt.Errorf("unipay.CreateOrder err: %s", err.Error())
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to create order by unipay")
		return fmt.Errorf("unipay.CreateOrder err: %s", err.Error())
	}
	log.Info("doAutoOrderCreate:", res.OrderId, res.PaymentAddress, res.ContractAddress, amount)

	orderInfo := tables.OrderInfo{
		OrderId:           res.OrderId,
		ActionType:        req.ActionType,
		Account:           req.SubAccount,
		AccountId:         subAccountId,
		ParentAccountId:   parentAccountId,
		Years:             req.Years,
		AlgorithmId:       hexAddr.DasAlgorithmId,
		PayAddress:        hexAddr.AddressHex,
		TokenId:           string(req.TokenId),
		Amount:            amount,
		USDAmount:         usdAmount,
		CouponCode:        req.CouponCode,
		PayStatus:         tables.PayStatusUnpaid,
		OrderStatus:       tables.OrderStatusDefault,
		Timestamp:         now.UnixMilli(),
		SvrName:           config.Cfg.Slb.SvrName,
		PremiumPercentage: premiumPercentage,
		PremiumBase:       premiumBase,
		PremiumAmount:     premiumAmount,
	}

	var paymentInfo tables.PaymentInfo
	if req.TokenId == tables.TokenIdStripeUSD && res.StripePaymentIntentId != "" {
		paymentInfo = tables.PaymentInfo{
			PayHash:       res.StripePaymentIntentId,
			OrderId:       res.OrderId,
			PayHashStatus: tables.PayHashStatusPending,
			Timestamp:     time.Now().UnixMilli(),
		}
	}
	if req.TokenId == tables.TokenIdDp {
		amount = amount.Div(decimal.New(1, tokenPrice.Decimals))
	}

	if err = h.DbDao.CreateOrderInfoWithCoupon(orderInfo, paymentInfo, couponInfo); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to create order")
		return fmt.Errorf("CreateOrderInfo err: %s", err.Error())
	}

	var resp RespAutoOrderCreate
	resp.OrderId = res.OrderId
	resp.Amount = amount
	resp.PaymentAddress = res.PaymentAddress
	resp.ContractAddress = res.ContractAddress
	resp.ClientSecret = res.ClientSecret
	if amount.Equal(decimal.Zero) {
		resp.PaymentStatus = PaymentWithout
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func RoundAmount(amount decimal.Decimal, tokenId tables.TokenId) decimal.Decimal {
	switch tokenId {
	case tables.TokenIdEth, tables.TokenIdBnb, tables.TokenIdMatic:
		dec := decimal.New(1, 8)
		amount = amount.Div(dec).Ceil().Mul(dec)
	case tables.TokenIdCkb, tables.TokenIdDoge:
		dec := decimal.New(1, 4)
		amount = amount.Div(dec).Ceil().Mul(dec)
	case tables.TokenIdTrx, tables.TokenIdErc20USDT,
		tables.TokenIdBep20USDT, tables.TokenIdTrc20USDT:
		dec := decimal.New(1, 3)
		amount = amount.Div(dec).Ceil().Mul(dec)
	}
	return amount
}
