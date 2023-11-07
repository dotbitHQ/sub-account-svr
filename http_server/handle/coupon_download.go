package handle

import (
	"encoding/csv"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

type ReqCouponDownload struct {
	core.ChainTypeAddress
	Account   string `json:"account" binding:"required"`
	Cid       string `json:"cid" binding:"required"`
	Timestamp int64  `json:"timestamp" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

func (h *HttpHandle) CouponDownload(ctx *gin.Context) {
	var (
		funcName               = "CouponDownload"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponDownload
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	if err = h.doCouponDownload(ctx, &req, &apiResp); err != nil {
		log.Error("doCouponDownload err:", err.Error(), funcName, clientIp, ctx)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponDownload(ctx *gin.Context, req *ReqCouponDownload, apiResp *api_code.ApiResp) error {
	if time.Now().After(time.UnixMilli(req.Timestamp).Add(time.Minute * 5)) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "timestamp expired, valid for 5 minutes")
		return nil
	}

	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)

	signMsg := fmt.Sprintf("%s%s%d", req.Account, req.Cid, req.Timestamp)
	if ok, err := sign.VerifyPersonalSignature(common.Hex2Bytes(req.Signature), []byte(signMsg), address); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
		return nil
	} else if !ok {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "signature invalid")
		return nil
	}

	setInfo, err := h.DbDao.GetCouponSetInfo(req.Cid)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon set info")
		return nil
	}
	if setInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "cid no exist")
		return nil
	}

	accInfo, err := h.DbDao.GetAccountInfoByAccountId(setInfo.AccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		return nil
	}
	if accInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountNotExist, "parent account does not exist")
		return nil
	}
	if !strings.EqualFold(accInfo.Manager, address) && !strings.EqualFold(accInfo.Owner, address) {
		apiResp.ApiRespErr(api_code.ApiCodeNoAccountPermissions, "no account permissions")
		return nil
	}

	couponList, err := h.DbDao.FindCouponCode(req.Cid)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon code")
		return nil
	}

	if len(couponList) == 0 {
		return nil
	}

	items := [][]string{
		{"code", "status"},
	}
	for _, v := range couponList {
		items = append(items, []string{v.Code, fmt.Sprint(v.Status)})
	}

	ctx.Header("Content-Type", "text/csv")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment;filename=%s.csv", req.Cid))
	wr := csv.NewWriter(ctx.Writer)
	if err := wr.WriteAll(items); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return nil
	}
	wr.Flush()
	return nil
}
