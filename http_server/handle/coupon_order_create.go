package handle

import (
	"context"
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
	OrderId   string         `json:"order_id"`
	Account   string         `json:"account" binding:"required"`
	TokenId   tables.TokenId `json:"token_id" binding:"required"`
	Num       int64          `json:"num" binding:"min=1,max=10000"`
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
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx.Request.Context())

	if err = h.doCouponOrderCreate(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doCouponOrderCreate err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponOrderCreate(ctx context.Context, req *ReqCouponOrderCreate, apiResp *api_code.ApiResp) error {
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
			log.Error(ctx, "UnLock err:", err.Error())
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

	hexAddr, err := req.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}

	var oldOrder tables.OrderInfo
	if req.OrderId != "" {
		oldOrder, err = h.DbDao.GetOrderByOrderID(req.OrderId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to get order")
			return err
		}
		oldOrder.OrderStatus = tables.OrderStatusClosed
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
	log.Info(ctx, "doCouponOrderCreate:", createOrderRes.OrderId, createOrderRes.PaymentAddress, createOrderRes.ContractAddress, amount)

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

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "price invalid")
		return nil
	}
	setInfo := tables.CouponSetInfo{
		OrderId:       createOrderRes.OrderId,
		AccountId:     res.accId,
		Account:       req.Account,
		ManagerAid:    int(hexAddr.DasAlgorithmId),
		ManagerSubAid: int(hexAddr.DasSubAlgorithmId),
		Manager:       hexAddr.AddressHex,
		Name:          req.Name,
		Note:          req.Note,
		Price:         price,
		Num:           req.Num,
		BeginAt:       req.BeginAt,
		ExpiredAt:     req.ExpiredAt,
		Status:        tables.CouponSetInfoStatusCreated,
	}
	setInfo.InitCid()

	if err = h.DbDao.CreateOrderInfo(orderInfo, oldOrder, paymentInfo, setInfo); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to create order")
		return fmt.Errorf("CreateOrderInfo err: %s", err.Error())
	}

	if req.TokenId == tables.TokenIdDp {
		amount = usdAmount
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

	if req.OrderId != "" {
		order, err := h.DbDao.GetOrderByOrderID(req.OrderId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to get order")
			return nil
		}
		if order.Id == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, "order does not exist")
			return nil
		}
		if order.OrderStatus == tables.OrderStatusSuccess ||
			order.OrderStatus == tables.OrderStatusFail ||
			order.OrderStatus == tables.OrderStatusClosed {
			apiResp.ApiRespErr(api_code.ApiCodeOrderClosed, "order closed")
			return nil
		}
	}

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

	if !strings.EqualFold(accInfo.Owner, address) && !strings.EqualFold(accInfo.Manager, address) {
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
	return &checkCreateParamsResp{
		accId:      accountId,
		dasAddr:    res,
		tokenPrice: tokenPrice,
	}
}
