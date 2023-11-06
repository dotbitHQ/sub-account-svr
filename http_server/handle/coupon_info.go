package handle

import (
	"das_sub_account/tables"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ReqCouponInfo struct {
	core.ChainTypeAddress
	Code string `json:"code"`
}

type RespCouponInfo struct {
	Code      string              `json:"code"`
	Price     string              `json:"price"`
	BeginAt   int64               `json:"begin_at"`
	ExpiredAt int64               `json:"expired_at"`
	Status    tables.CouponStatus `json:"status"`
}

func (h *HttpHandle) CouponInfo(ctx *gin.Context) {
	var (
		funcName               = "CouponInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponInfo
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx)

	if err = ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ctx.ShouldBindJSON err:", err.Error(), funcName, clientIp, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	if err = h.doCouponInfo(&req, &apiResp); err != nil {
		log.Error("doCouponInfo err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponInfo(req *ReqCouponInfo, apiResp *api_code.ApiResp) error {
	couponInfo, err := h.DbDao.GetCouponByCode(req.Code)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "coupon info find failed")
		return err
	}
	if couponInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "code no exist")
		return nil
	}
	setInfo, err := h.DbDao.GetCouponSetInfo(couponInfo.Cid)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "coupon set info find failed")
		return err
	}
	if setInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "coupon set info no exist")
		return nil
	}

	resp := &RespCouponInfo{
		Code:      req.Code,
		Price:     setInfo.Price.String(),
		BeginAt:   setInfo.BeginAt,
		ExpiredAt: setInfo.ExpiredAt,
		Status:    couponInfo.Status,
	}
	apiResp.ApiRespOK(resp)
	return nil
}
