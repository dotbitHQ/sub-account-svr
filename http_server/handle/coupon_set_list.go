package handle

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type ReqCouponSetList struct {
	core.ChainTypeAddress
	Account  string `json:"account"`
	Page     int    `json:"page" binding:"gte=1"`
	PageSize int    `json:"page_size" binding:"gte=1,lte=100"`
}

type RespCouponSetInfoList struct {
	Total int64               `json:"total"`
	List  []RespCouponSetInfo `json:"list"`
}

type RespCouponSetInfo struct {
	Cid       string `json:"cid"`
	OrderId   string `json:"order_id"`
	Account   string `json:"account" `
	Name      string `json:"name"`
	Note      string `json:"note"`
	Price     string `json:"price" `
	Num       int    `json:"num" `
	ExpiredAt int64  `json:"expired_at"`
	CreatedAt int64  `json:"created_at"`
}

func (h *HttpHandle) CouponSetList(ctx *gin.Context) {
	var (
		funcName               = "CouponSetList"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponSetList
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx)

	if err = h.doCouponSetList(&req, &apiResp); err != nil {
		log.Error("doCouponSetList err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponSetList(req *ReqCouponSetList, apiResp *api_code.ApiResp) error {
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
	setInfo, total, err := h.DbDao.FindCouponSetInfoList(accountId, req.Page, req.PageSize)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return nil
	}

	resp := &RespCouponSetInfoList{
		Total: total,
		List:  make([]RespCouponSetInfo, 0),
	}
	for _, v := range setInfo {
		resp.List = append(resp.List, RespCouponSetInfo{
			Cid:       v.Cid,
			OrderId:   v.OrderId,
			Account:   v.Account,
			Name:      v.Name,
			Note:      v.Note,
			Price:     v.Price,
			Num:       v.Num,
			ExpiredAt: v.ExpiredAt,
			CreatedAt: v.CreatedAt.UnixMilli(),
		})
	}

	apiResp.ApiRespOK(resp)
	return nil
}
