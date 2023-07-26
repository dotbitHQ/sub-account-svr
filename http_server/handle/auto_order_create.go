package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"das_sub_account/unipay"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"strings"
	"time"
)

type ReqAutoOrderCreate struct {
	core.ChainTypeAddress
	ActionType tables.ActionType `json:"action_type"`
	SubAccount string            `json:"sub_account"`
	TokenId    tables.TokenId    `json:"token_id"`
	Years      uint64            `json:"years"`
}

type RespAutoOrderCreate struct {
	OrderId         string          `json:"order_id"`
	PaymentAddress  string          `json:"payment_address"`
	ContractAddress string          `json:"contract_address"`
	ClientSecret    string          `json:"client_secret"`
	Amount          decimal.Decimal `json:"amount"`
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
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

	if err = h.doAutoOrderCreate(&req, &apiResp); err != nil {
		log.Error("doAutoOrderCreate err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoOrderCreate(req *ReqAutoOrderCreate, apiResp *api_code.ApiResp) error {
	var resp RespAutoOrderCreate
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
	err = h.checkSwitch(parentAccountId, apiResp)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check token id
	userConfig, err := h.DbDao.GetUserPaymentConfig(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to search payment config")
		return fmt.Errorf("GetUserPaymentConfig err: %s", err.Error())
	} else if cfg, ok := userConfig.CfgMap[string(req.TokenId)]; !ok || !cfg.Enable {
		apiResp.ApiRespErr(api_code.ApiCodeTokenIdNotSupported, "payment method not supported")
		return nil
	}

	// get max years
	if maxYear := h.getMaxYears(parentAccount); req.Years > maxYear {
		apiResp.ApiRespErr(api_code.ApiCodeBeyondMaxYears, "The main account is valid for less than one year")
		return nil
	}

	// get rule price
	usdAmount, err := h.getRulePrice(parentAccount.Account, parentAccountId, req.SubAccount, apiResp)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	tokenPrice, err := h.DbDao.GetTokenById(req.TokenId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to search token price")
		return fmt.Errorf("GetTokenById err: %s", err.Error())
	} else if tokenPrice.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeTokenIdNotSupported, "payment method not supported")
		return nil
	}
	log.Info("usdAmount:", usdAmount.String(), req.Years)
	usdAmount = usdAmount.Mul(decimal.NewFromInt(int64(req.Years)))
	//log.Info("usdAmount:", usdAmount.String(), req.Years)
	amount := usdAmount.Mul(decimal.New(1, tokenPrice.Decimals)).Div(tokenPrice.Price).Ceil()
	if amount.Cmp(decimal.Zero) != 1 {
		apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("price err: %s", amount.String()))
		return nil
	}
	//
	//if req.TokenId == tables.TokenIdStripeUSD && amount.Cmp(decimal.NewFromFloat(0.52)) == -1 {
	//	apiResp.ApiRespErr(http_api.ApiCodeAmountIsTooLow, "Prices must not be lower than 0.52$")
	//	return nil
	//}
	// create order
	premiumPercentage := decimal.Zero
	premiumBase := decimal.Zero
	if req.TokenId == tables.TokenIdStripeUSD {
		premiumPercentage = config.Cfg.Stripe.PremiumPercentage
		premiumBase = config.Cfg.Stripe.PremiumBase
		amount = amount.Mul(premiumPercentage.Add(decimal.NewFromInt(1))).Add(premiumBase.Mul(decimal.NewFromInt(100)))
		amount = decimal.NewFromInt(amount.IntPart())
		usdAmount = usdAmount.Mul(premiumPercentage.Add(decimal.NewFromInt(1))).Add(premiumBase.Mul(decimal.NewFromInt(100)))
	}

	res, err := unipay.CreateOrder(unipay.ReqOrderCreate{
		ChainTypeAddress:  req.ChainTypeAddress,
		BusinessId:        unipay.BusinessIdAutoSubAccount,
		Amount:            amount,
		PayTokenId:        req.TokenId,
		PaymentAddress:    config.GetUnipayAddress(req.TokenId),
		PremiumPercentage: premiumPercentage,
		PremiumBase:       premiumBase,
	})
	if err != nil {
		if strings.Contains(err.Error(), "600004") {
			apiResp.ApiRespErr(http_api.ApiCodePaymentMethodDisable, "This payment method is unavailable")
			return fmt.Errorf("unipay.CreateOrder err: %s", err.Error())
		}
		if strings.Contains(err.Error(), "600003") {
			apiResp.ApiRespErr(http_api.ApiCodeAmountIsTooLow, "Amount must not be lower than 0.52$")
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
		PayStatus:         tables.PayStatusUnpaid,
		OrderStatus:       tables.OrderStatusDefault,
		Timestamp:         time.Now().UnixMilli(),
		SvrName:           config.Cfg.Slb.SvrName,
		PremiumPercentage: premiumPercentage,
		PremiumBase:       premiumBase,
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
	if err = h.DbDao.CreateOrderInfo(orderInfo, paymentInfo); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to create order")
		return fmt.Errorf("CreateOrderInfo err: %s", err.Error())
	}

	resp.OrderId = res.OrderId
	resp.Amount = amount
	resp.PaymentAddress = res.PaymentAddress
	resp.ContractAddress = res.ContractAddress
	resp.ClientSecret = res.ClientSecret

	apiResp.ApiRespOK(resp)
	return nil
}
