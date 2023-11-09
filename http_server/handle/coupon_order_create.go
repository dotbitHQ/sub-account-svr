package handle

import (
	"crypto/md5"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"das_sub_account/unipay"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const couponCreateLockKey = "coupon:create:"

var (
	priceReg = regexp.MustCompile(`^(\d+)(.\d{0,2})?$`)
)

type ReqCouponOrderCreate struct {
	core.ChainTypeAddress
	Account   string         `json:"account" binding:"required"`
	TokenId   tables.TokenId `json:"token_id" binding:"required"`
	Num       int            `json:"num" binding:"min=1,max=10000"`
	Cid       string         `json:"cid" binding:"required"`
	Name      string         `json:"name" binding:"required"`
	Note      string         `json:"note"`
	Price     string         `json:"price" binding:"required"`
	BeginAt   int64          `json:"begin_at"`
	ExpiredAt int64          `json:"expired_at" binding:"required"`
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

	if err := ctx.ShouldBindBodyWith(&req, binding.JSON); err != nil {
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
	lockKey := fmt.Sprintf("%x", md5.Sum([]byte(couponCreateLockKey+req.Account)))
	if err := h.RC.Lock(lockKey); err != nil {
		if errors.Is(err, cache.ErrDistributedLockPreemption) {
			apiResp.ApiRespErr(api_code.ApiCodeOperationFrequent, "request too frequent")
			return nil
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to lock")
		return nil
	}
	defer func() {
		if err := h.RC.UnLock(lockKey); err != nil {
			log.Error("UnLock err:", err.Error())
		}
	}()

	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	res := h.couponCreateParamsCheck(req, apiResp)
	if apiResp.ErrNo != 0 {
		return nil
	}

	usdAmount := decimal.NewFromFloat(config.Cfg.Das.Coupon.CouponPrice).Mul(decimal.NewFromInt(int64(req.Num)))
	amount := usdAmount.Mul(decimal.New(1, res.tokenPrice.Decimals)).Div(res.tokenPrice.Price).Ceil()
	if amount.LessThanOrEqual(decimal.Zero) {
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

	order, err := h.DbDao.GetPendingOrderByAccIdAndActionType(res.accId, tables.ActionTypeCouponCreate)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to get pending order")
		return fmt.Errorf("GetPendingOrderByAccIdAndActionType err: %s", err.Error())
	}
	if order.Id > 0 {
		apiResp.ApiRespErr(api_code.ApiCodeOperationFrequent, "have pending order: "+order.OrderId)
		apiResp.Data = map[string]interface{}{
			"order_id": order.OrderId,
		}
		return nil
	}

	hexAddr, err := req.FormatChainTypeAddress(h.DasCore.NetType(), false)
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
			"algorithm_id": fmt.Sprint(hexAddr.DasAlgorithmId),
			"address":      hexAddr.AddressHex,
			"action":       "coupon_create",
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
		AccountId:         res.accId,
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
	}
	reqData, _ := json.Marshal(req)
	orderInfo.MetaData = string(reqData)

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

type checkCreateParamsResp struct {
	accId      string
	dasAddr    *core.DasAddressHex
	tokenPrice tables.TTokenPriceInfo
}

func (h *HttpHandle) couponCreateParamsCheck(req *ReqCouponOrderCreate, apiResp *api_code.ApiResp) *checkCreateParamsResp {
	log.Infof("couponCreateParamsCheck: %+v", *req)

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	accInfo, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "get account info failed")
		return nil
	}
	if accInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account does not exist")
		return nil
	}
	if accInfo.ParentAccountId != "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "sub account cannot create coupon")
		return nil
	}
	if accInfo.Status != tables.AccountStatusNormal {
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusNotNormal, "account status is not normal")
		return nil
	}
	if accInfo.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "account expired")
		return nil
	}
	if !priceReg.MatchString(req.Price) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "price invalid")
		return nil
	}
	price, _ := strconv.ParseFloat(req.Price, 64)
	if price < config.Cfg.Das.Coupon.PriceMin || price > config.Cfg.Das.Coupon.PriceMax {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "price invalid")
		return nil
	}
	if req.BeginAt >= req.ExpiredAt {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "begin time must less than expired time")
		return nil
	}

	if time.UnixMilli(req.ExpiredAt).Before(time.Now()) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "expired_at invalid")
		return nil
	}

	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)

	if !strings.EqualFold(accInfo.Manager, address) {
		apiResp.ApiRespErr(api_code.ApiCodeNoAccountPermissions, "no account permissions")
		return nil
	}

	tokenPrice, err := h.DbDao.GetTokenById(req.TokenId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to search token price")
		return nil
	} else if tokenPrice.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeTokenIdNotSupported, "payment method not supported")
		return nil
	}

	unpaidSetInfo, err := h.DbDao.GetUnPaidCouponSetByAccId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return nil
	}
	if unpaidSetInfo.Id > 0 {
		apiResp.ApiRespErr(api_code.ApiCodeCouponUnpaid, "have unpaid coupon order")
		apiResp.Data = map[string]interface{}{
			"cid": unpaidSetInfo.Cid,
		}
		return nil
	}
	return &checkCreateParamsResp{
		accId:      accountId,
		dasAddr:    res,
		tokenPrice: tokenPrice,
	}
}
