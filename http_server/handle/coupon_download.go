package handle

import (
	"das_sub_account/tables"
	"encoding/csv"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"strings"
)

type ReqCouponDownload struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
	Cid     string `json:"cid" binding:"required"`
}

func (h *HttpHandle) CouponDownload(ctx *gin.Context) {
	var (
		funcName               = "CouponDownload"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCouponDownload
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

	if err = h.doCouponDownload(ctx, &req, &apiResp); err != nil {
		log.Error("doCouponDownload err:", err.Error(), funcName, clientIp, ctx.Request.Context())
	}
}

func (h *HttpHandle) doCouponDownload(ctx *gin.Context, req *ReqCouponDownload, apiResp *api_code.ApiResp) error {
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
	accName := strings.TrimSuffix(req.Account, common.DasAccountSuffix)

	couponList, err := h.DbDao.FindCouponCode(req.Cid)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query coupon code")
		return nil
	}
	if len(couponList) == 0 {
		return nil
	}

	topDidLink := "https://test.topdid.com"
	if h.DasCore.NetType() == common.DasNetTypeMainNet {
		topDidLink = "https://topdid.com"
	}

	items := [][]string{
		{"Coupon Code", "Status", "Used For", "Link"},
	}
	for _, v := range couponList {
		item := []string{v.Code}
		if v.Status == tables.CouponStatusNormal {
			item = append(item, "Available")
			item = append(item, "-")
			item = append(item, fmt.Sprintf("%s/mint/.%s?coupon_code=%s", topDidLink, accName, v.Code))
		} else {
			item = append(item, "Used")
			order, err := h.DbDao.GetOrderByCoupon(v.Code)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order info")
				return err
			}
			item = append(item, strings.TrimSuffix(order.Account, common.DasAccountSuffix))
			item = append(item, "-")
		}
		items = append(items, item)
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment;filename=%s-%s.csv", accName, setInfo.Name))
	ctx.Header("Content-Transfer-Encoding", "binary")
	wr := csv.NewWriter(ctx.Writer)
	if err := wr.WriteAll(items); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return nil
	}
	wr.Flush()
	ctx.Status(http.StatusOK)
	return nil
}
