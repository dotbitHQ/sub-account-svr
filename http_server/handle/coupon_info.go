package handle

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"net/http"
	"time"
)

var couponInfoLockLua = `
	local key = KEYS[1]
	local code = KEYS[2]
	redis.call('HINCRBY', key, code, 1)
	redis.call('HINCRBY', key, 'total', 1)
	redis.call('EXPIREAT', key, ARGV[1])
`

type ReqCouponInfo struct {
	core.ChainTypeAddress
	Code     string `json:"code" binding:"required"`
	clientIP string
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
	req.clientIP = clientIp

	if err = h.doCouponInfo(&req, &apiResp); err != nil {
		log.Error("doCouponInfo err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponInfo(req *ReqCouponInfo, apiResp *api_code.ApiResp) error {
	lockKey := fmt.Sprintf("coupon_info:%s", req.clientIP)
	if _, err := h.RC.Red.Pipelined(func(p redis.Pipeliner) error {
		p.HIncrBy(lockKey, req.Code, 1)
		p.ExpireAt(lockKey, time.Now().Add(time.Second*10))
		return nil
	}); err != nil {
		return err
	}
	if h.RC.Red.HLen(lockKey).Val() > 2 {
		apiResp.ApiRespErr(api_code.ApiCodeOperationFrequent, "operation frequent")
		return nil
	}

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
