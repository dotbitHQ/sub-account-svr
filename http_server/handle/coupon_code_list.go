package handle

import (
	"context"
	"das_sub_account/tables"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
)

type ReqCouponCodeList struct {
	core.ChainTypeAddress
	Account  string `json:"account" binding:"required"`
	Cid      string `json:"cid" binding:"required"`
	Page     int    `json:"page" binding:"gte=1"`
	PageSize int    `json:"page_size" binding:"gte=1,lte=100"`
}

type RespCouponCodeList struct {
	Total     int64            `json:"total"`
	Used      int64            `json:"used"`
	Name      string           `json:"name"`
	Note      string           `json:"note"`
	Price     string           `json:"price"`
	BeginAt   int64            `json:"begin_at"`
	ExpiredAt int64            `json:"expired_at"`
	CreatedAt int64            `json:"created_at"`
	List      []RespCouponCode `json:"list"`
}

type RespCouponCode struct {
	Code   string              `json:"code"`
	UsedBy string              `json:"used_by"`
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
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx.Request.Context())

	if err := ctx.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	if err = h.doCouponCodeList(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doCouponCodeList err:", err.Error(), funcName, clientIp, ctx.Request.Context())
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponCodeList(ctx context.Context, req *ReqCouponCodeList, apiResp *api_code.ApiResp) error {
	setInfo, err := h.DbDao.GetCouponSetInfo(req.Cid)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon set info")
		return nil
	}
	if setInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "cid no exist")
		return nil
	}

	// get coupon set list
	couponList, total, used, err := h.DbDao.FindCouponCodeList(req.Cid, req.Page, req.PageSize)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return nil
	}

	resp := &RespCouponCodeList{
		Total:     total,
		Used:      used,
		Name:      setInfo.Name,
		Note:      setInfo.Note,
		Price:     setInfo.Price.String(),
		BeginAt:   setInfo.BeginAt,
		ExpiredAt: setInfo.ExpiredAt,
		CreatedAt: setInfo.CreatedAt.UnixMilli(),
		List:      make([]RespCouponCode, 0),
	}
	for _, v := range couponList {
		couponInfo := RespCouponCode{
			Code:   v.Code,
			Status: v.Status,
		}
		if v.Status == tables.CouponStatusUsed {
			order, err := h.DbDao.GetOrderByCoupon(v.Code)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
				return nil
			}
			if order.Id == 0 {
				apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, "order not exist")
				return nil
			}
			couponInfo.UsedBy = order.Account
		}
		resp.List = append(resp.List, couponInfo)
	}
	apiResp.ApiRespOK(resp)
	return nil
}
