package handle

import (
	"crypto/md5"
	"das_sub_account/config"
	"das_sub_account/consts"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/random"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	priceReg = regexp.MustCompile(`^(\d+)(.\d{0,2})?$`)
)

type ReqCouponCreate struct {
	core.ChainTypeAddress
	Account   string `json:"account" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Note      string `json:"note"`
	Price     string `json:"price" binding:"required"`
	Num       int    `json:"num" binding:"min=1"`
	ExpiredAt int64  `json:"expired_at" binding:"required"`
	Timestamp int64  `json:"timestamp" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

type RespCouponCreate struct {
	SignInfoList
	CouponCode []string `json:"coupon_code"`
}

type CouponCreateSignCache struct {
	ReqCouponCreate
	CouponCode []string `json:"coupon_code"`
}

func (r *CouponCreateSignCache) GetSignInfo() (signKey, reqDataStr string) {
	reqData, _ := json.Marshal(r)
	reqDataStr = string(reqData)
	signKey = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s_%d", reqDataStr, time.Now().UnixNano()))))
	return
}

func (h *HttpHandle) CouponCreate(ctx *gin.Context) {
	var (
		funcName               = "CouponCreate"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponCreate
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx)

	if err = h.doCouponCreate(&req, &apiResp); err != nil {
		log.Error("doCouponCreate err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponCreate(req *ReqCouponCreate, apiResp *api_code.ApiResp) error {
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

	resp := &RespCouponCreate{
		CouponCode: make([]string, 0),
	}
	couponCodes := make(map[string]struct{})
	for {
		h.createCoupon(couponCodes, req)
		h.DbDao.CouponExists()
		h.DbDao.FindTokens()
	}

	kvs := make([]smt.SmtKv, 0)
	for code := range couponCodes {
		kvs = append(kvs, smt.SmtKv{
			Key:   smt.Sha256(code),
			Value: smt.Sha256(code),
		})
	}

	tree := smt.NewSmtSrv(*h.SmtServerUrl, "")
	smtOut, err := tree.UpdateSmt(kvs, smt.SmtOpt{GetRoot: true})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return err
	}
	signMsg := fmt.Sprintf("%s%s", common.DotBitPrefix, smtOut.Root.String())

	signCache := &CouponCreateSignCache{
		ReqCouponCreate: *req,
		CouponCode:      resp.CouponCode,
	}

	signKey, reqDataStr := signCache.GetSignInfo()
	if err := h.RC.SetSignTxCache(signKey, reqDataStr); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	signType := res.DasAlgorithmId
	if signType == common.DasAlgorithmIdEth712 {
		signType = common.DasAlgorithmIdEth
	}

	resp.Action = consts.ActionCouponCreate
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		SignList: []txbuilder.SignData{{
			SignType: signType,
			SignMsg:  signMsg,
		}},
	})
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) couponCreateParamsCheck(req *ReqCouponCreate, apiResp *api_code.ApiResp) *core.DasAddressHex {
	if time.UnixMilli(req.Timestamp).Add(time.Minute * 10).Before(time.Now()) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params timestamp invalid")
		return nil
	}

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	accInfo, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		return nil
	}
	if accInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountNotExist, "parent account does not exist")
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

	if !strings.EqualFold(accInfo.Manager, address) ||
		accInfo.ManagerAlgorithmId != res.DasAlgorithmId ||
		accInfo.ManagerSubAid != res.DasSubAlgorithmId {
		apiResp.ApiRespErr(api_code.ApiCodeNoAccountPermissions, "no account permissions")
		return nil
	}
	return res
}

func (h *HttpHandle) createCoupon(couponCodes map[string]struct{}, req *ReqCouponCreate) {
	for {
		md5Res := md5.Sum([]byte(fmt.Sprintf("%s%d%d%s", req.Price, time.Now().UnixNano(), req.ExpiredAt, random.String(8, random.Alphanumeric))))
		base58Res := base58.Encode([]byte(fmt.Sprintf("%x", md5Res)))
		code := strings.ToUpper(base58Res[:8])
		if _, ok := couponCodes[code]; ok {
			continue
		}
		couponCodes[code] = struct{}{}

		if len(couponCodes) >= req.Num {
			break
		}
	}
}
