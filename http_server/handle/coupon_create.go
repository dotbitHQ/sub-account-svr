package handle

import (
	"das_sub_account/http_server/api_code"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ReqCouponCreate struct {
	core.ChainTypeAddress
	Account   string `json:"account" binding:"required"`
	Name      string `json:"name"`
	Note      string `json:"note"`
	Price     string `json:"price" binding:"required"`
	Num       int    `json:"num" binding:"min=1"`
	ExpiredAt int64  `json:"expired_at"`
	Signature string `json:"signature" binding:"required"`
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

	return nil
}
