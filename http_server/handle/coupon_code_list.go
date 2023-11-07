package handle

import (
	"das_sub_account/tables"
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

type ReqCouponCodeList struct {
	core.ChainTypeAddress
	Account   string `json:"account" binding:"required"`
	Cid       string `json:"cid" binding:"required"`
	Timestamp int64  `json:"timestamp" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	Page      int    `json:"page" binding:"gte=1"`
	PageSize  int    `json:"page_size" binding:"gte=1,lte=100"`
}

type RespCouponCodeList struct {
	Total     int64            `json:"total"`
	Cid       string           `json:"cid"`
	OrderId   string           `json:"order_id"`
	Account   string           `json:"account" `
	Name      string           `json:"name"`
	Note      string           `json:"note"`
	Price     string           `json:"price"`
	Num       int              `json:"num"`
	Status    int              `json:"status"`
	BeginAt   int64            `json:"begin_at"`
	ExpiredAt int64            `json:"expired_at"`
	CreatedAt int64            `json:"created_at"`
	List      []RespCouponCode `json:"list"`
}

type RespCouponCode struct {
	Code   string              `json:"code"`
	Status tables.CouponStatus `json:"status"`
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

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	if err = h.doCouponCodeList(&req, &apiResp); err != nil {
		log.Error("doCouponCodeList err:", err.Error(), funcName, clientIp, ctx)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponCodeList(req *ReqCouponCodeList, apiResp *api_code.ApiResp) error {
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

	if !strings.EqualFold(accInfo.Manager, address) {
		apiResp.ApiRespErr(api_code.ApiCodeNoAccountPermissions, "no account permissions")
		return nil
	}

	// get coupon set list
	couponList, total, err := h.DbDao.FindCouponCodeList(req.Cid, req.Page, req.PageSize)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return nil
	}

	resp := &RespCouponCodeList{
		Total:     total,
		Cid:       setInfo.Cid,
		OrderId:   setInfo.OrderId,
		Account:   setInfo.Account,
		Name:      setInfo.Name,
		Note:      setInfo.Note,
		Price:     setInfo.Price.String(),
		Num:       setInfo.Num,
		Status:    setInfo.Status,
		BeginAt:   setInfo.BeginAt,
		ExpiredAt: setInfo.ExpiredAt,
		CreatedAt: setInfo.CreatedAt.UnixMilli(),
		List:      make([]RespCouponCode, 0),
	}
	for _, v := range couponList {
		resp.List = append(resp.List, RespCouponCode{
			Code:   v.Code,
			Status: v.Status,
		})
	}
	apiResp.ApiRespOK(resp)
	return nil
}
