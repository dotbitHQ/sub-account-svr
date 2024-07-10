package handle

import (
	"context"
	"das_sub_account/tables"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ReqCouponStatisticalInfo struct {
}

type RespCouponStatisticalInfo struct {
	Total    int64 `json:"total"`
	Used     int64 `json:"used"`
	Accounts int64 `json:"accounts"`
}

func (h *HttpHandle) CouponStatisticalInfo(ctx *gin.Context) {
	var (
		funcName               = "CouponStatisticalInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponStatisticalInfo
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx.Request.Context())

	if err = ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ctx.ShouldBindJSON err:", err.Error(), funcName, clientIp, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	if err = h.doCouponStatisticalInfo(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doCouponStatisticalInfo err:", err.Error(), funcName, clientIp, ctx.Request.Context())
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponStatisticalInfo(ctx context.Context, req *ReqCouponStatisticalInfo, apiResp *api_code.ApiResp) error {
	total, err := h.DbDao.GetCouponNum()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon")
		return err
	}
	used, err := h.DbDao.GetCouponNum(tables.CouponStatusUsed)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon")
		return err
	}
	accounts, err := h.DbDao.GetCouponAccount()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon")
		return err
	}

	resp := &RespCouponStatisticalInfo{
		Total:    total,
		Used:     used,
		Accounts: accounts,
	}
	apiResp.ApiRespOK(resp)
	return nil
}
