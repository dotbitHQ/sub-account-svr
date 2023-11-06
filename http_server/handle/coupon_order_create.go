package handle

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"das_sub_account/unipay"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"strings"
	"time"
)

type ReqCouponOrderCreate struct {
	core.ChainTypeAddress
	Account string         `json:"account" binding:"required"`
	TokenId tables.TokenId `json:"token_id" binding:"required"`
	Num     int            `json:"num"`
	Cid     string         `json:"cid" binding:"required"`
}

type RespCouponOrderCreate struct {
	OrderId         string          `json:"order_id"`
	PaymentAddress  string          `json:"payment_address"`
	ContractAddress string          `json:"contract_address"`
	ClientSecret    string          `json:"client_secret"`
	Amount          decimal.Decimal `json:"amount"`
}

func (h *HttpHandle) CouponOrderCreate(ctx *gin.Context) {
	var (
		funcName               = "CouponOrderCreate"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponOrderCreate
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

	if err = h.doCouponOrderCreate(&req, &apiResp); err != nil {
		log.Error("doCouponOrderCreate err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponOrderCreate(req *ReqCouponOrderCreate, apiResp *api_code.ApiResp) error {
	accId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	// check parent account
	accountInfo, err := h.checkParentAccount(apiResp, accId)
	if err != nil {
		return err
	}
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)

	if !strings.EqualFold(accountInfo.Manager, address) && !strings.EqualFold(accountInfo.Owner, address) {
		apiResp.ApiRespErr(api_code.ApiCodeNoAccountPermissions, "no account permissions")
		return nil
	}

	// check cid
	setInfo, err := h.DbDao.GetCouponSetInfo(req.Cid)
	if err != nil {
		return err
	}
	if setInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeCouponCidNotExist, "coupon cid not exist")
		return nil
	}
	if setInfo.OrderId != "" {
		apiResp.ApiRespErr(api_code.ApiCodeCouponPaid, "coupon is paid, no need to pay twice")
		return nil
	}
	if setInfo.Num != req.Num {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("field 'num' must be: %d", setInfo.Num))
		return nil
	}
	if setInfo.Account != req.Account {
		apiResp.ApiRespErr(api_code.ApiCodeNoAccountPermissions, "no account permissions")
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

	usdAmount := decimal.NewFromFloat(config.Cfg.Das.Coupon.CouponPrice).Mul(decimal.NewFromInt(int64(req.Num)))
	amount := usdAmount.Mul(decimal.New(1, tokenPrice.Decimals)).Div(tokenPrice.Price).Ceil()
	if amount.Cmp(decimal.Zero) != 1 {
		apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("price err: %s", amount.String()))
		return nil
	}
	amount = RoundAmount(amount, req.TokenId)

	premiumPercentage := decimal.Zero
	premiumBase := decimal.Zero
	premiumAmount := decimal.Zero
	if req.TokenId == tables.TokenIdStripeUSD {
		premiumPercentage = config.Cfg.Stripe.PremiumPercentage
		premiumBase = config.Cfg.Stripe.PremiumBase
		premiumAmount = amount
		amount = amount.Mul(premiumPercentage.Add(decimal.NewFromInt(1))).Add(premiumBase.Mul(decimal.NewFromInt(100)))
		amount = decimal.NewFromInt(amount.Ceil().IntPart())
		premiumAmount = amount.Sub(premiumAmount)
		usdAmount = usdAmount.Mul(premiumPercentage.Add(decimal.NewFromInt(1))).Add(premiumBase.Mul(decimal.NewFromInt(100)))
	}

	hexAddr, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}

	createOrderRes, err := unipay.CreateOrder(unipay.ReqOrderCreate{
		ChainTypeAddress:  req.ChainTypeAddress,
		BusinessId:        unipay.BusinessIdAutoSubAccount,
		Amount:            amount,
		PayTokenId:        req.TokenId,
		PaymentAddress:    config.GetUnipayAddress(req.TokenId),
		PremiumPercentage: premiumPercentage,
		PremiumBase:       premiumBase,
		PremiumAmount:     premiumAmount,
		MetaData: map[string]string{
			"account":      req.Account,
			"cid":          req.Cid,
			"algorithm_id": fmt.Sprint(hexAddr.DasAlgorithmId),
			"address":      hexAddr.AddressHex,
			"action":       "create",
		},
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to create order by unipay")
		return fmt.Errorf("unipay.CreateOrder err: %s", err.Error())
	}
	log.Info("doCouponOrderCreate:", createOrderRes.OrderId, createOrderRes.PaymentAddress, createOrderRes.ContractAddress, amount)

	orderInfo := tables.OrderInfo{
		OrderId:           createOrderRes.OrderId,
		ActionType:        tables.ActionTypeCouponCreate,
		Account:           req.Account,
		AccountId:         accId,
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
		PremiumAmount:     premiumAmount,
		MetaData: &tables.MetaData{
			Cid: req.Cid,
		},
	}

	var paymentInfo tables.PaymentInfo
	if req.TokenId == tables.TokenIdStripeUSD && createOrderRes.StripePaymentIntentId != "" {
		paymentInfo = tables.PaymentInfo{
			PayHash:       createOrderRes.StripePaymentIntentId,
			OrderId:       createOrderRes.OrderId,
			PayHashStatus: tables.PayHashStatusPending,
			Timestamp:     time.Now().UnixMilli(),
		}
	}
	if err = h.DbDao.CreateOrderInfo(orderInfo, paymentInfo); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to create order")
		return fmt.Errorf("CreateOrderInfo err: %s", err.Error())
	}

	var resp RespCouponOrderCreate
	resp.OrderId = createOrderRes.OrderId
	resp.Amount = amount
	resp.PaymentAddress = createOrderRes.PaymentAddress
	resp.ContractAddress = createOrderRes.ContractAddress
	resp.ClientSecret = createOrderRes.ClientSecret

	apiResp.ApiRespOK(resp)
	return nil
}
