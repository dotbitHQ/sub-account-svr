package handle

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type ReqCouponCodeList struct {
	core.ChainTypeAddress
	Cid      string `json:"cid"`
	Page     int    `json:"page" binding:"gte=1"`
	PageSize int    `json:"page_size" binding:"gte=1,lte=100"`
}

type RespCouponCodeList struct {
	Total int64            `json:"total"`
	List  []RespCouponCode `json:"list"`
}

type RespCouponCode struct {
	Code   string `json:"code"`
	Status int    `json:"status"`
}

func (h *HttpHandle) CouponCodeList(ctx *gin.Context) {
	var (
		funcName               = "CouponCodeList"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponCodeList
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx)

	if err = h.doCouponCodeList(&req, &apiResp); err != nil {
		log.Error("doCouponCodeList err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponCodeList(req *ReqCouponCodeList, apiResp *api_code.ApiResp) error {
	couponSetInfo, err := h.DbDao.GetCouponSetInfo(req.Cid)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon set info")
		return nil
	}
	if couponSetInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "cid no exist")
		return nil
	}

	accInfo, err := h.DbDao.GetAccountInfoByAccountId(couponSetInfo.AccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		return nil
	}
	if accInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountNotExist, "parent account does not exist")
		return nil
	}

	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)

	if (!strings.EqualFold(accInfo.Manager, address) ||
		accInfo.ManagerAlgorithmId != res.DasAlgorithmId ||
		accInfo.ManagerSubAid != res.DasSubAlgorithmId) &&
		(!strings.EqualFold(accInfo.Owner, address) ||
			accInfo.OwnerAlgorithmId != res.DasAlgorithmId ||
			accInfo.OwnerSubAid != res.DasSubAlgorithmId) {
		apiResp.ApiRespErr(api_code.ApiCodeNoAccountPermissions, "no account permissions")
		return nil
	}

	// get coupon set list
	setInfo, total, err := h.DbDao.FindCouponCodeList(req.Cid, req.Page, req.PageSize)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return nil
	}

	resp := &RespCouponCodeList{
		Total: total,
		List:  make([]RespCouponCode, 0),
	}
	for _, v := range setInfo {
		resp.List = append(resp.List, RespCouponCode{
			Code:   string(*v.Code),
			Status: v.Status,
		})
	}
	apiResp.ApiRespOK(resp)
	return nil
}
