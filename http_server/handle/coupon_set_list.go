package handle

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"net/http"
)

type ReqCouponSetList struct {
	core.ChainTypeAddress
	Account  string `json:"account" binding:"required"`
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
	Price     string `json:"price"`
	Num       int64  `json:"num"`
	Used      int64  `json:"used"`
	Status    int    `json:"status"`
	BeginAt   int64  `json:"begin_at"`
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

	if err = ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ctx.ShouldBindJSON err:", err.Error(), funcName, clientIp, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	if err = h.doCouponSetList(&req, &apiResp); err != nil {
		log.Error("doCouponSetList err:", err.Error(), funcName, clientIp, ctx)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCouponSetList(req *ReqCouponSetList, apiResp *api_code.ApiResp) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	setInfo, total, err := h.DbDao.FindCouponSetInfoList(accountId, req.Page, req.PageSize)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return nil
	}

	resp := &RespCouponSetInfoList{
		Total: total,
		List:  make([]RespCouponSetInfo, 0, len(setInfo)),
	}

	errWg := &errgroup.Group{}
	ch := make(chan int, 10)
	errWg.Go(func() error {
		for idx := range setInfo {
			ch <- idx
		}
		close(ch)
		return nil
	})

	errWg.Go(func() error {
		for idx := range ch {
			used, err := h.DbDao.GetUsedCoupon(setInfo[idx].Cid)
			if err != nil {
				return err
			}
			v := setInfo[idx]
			resp.List = append(resp.List, RespCouponSetInfo{
				Cid:       v.Cid,
				OrderId:   v.OrderId,
				Account:   v.Account,
				Name:      v.Name,
				Note:      v.Note,
				Price:     v.Price.String(),
				Num:       v.Num,
				Used:      used,
				Status:    v.Status,
				BeginAt:   v.BeginAt,
				ExpiredAt: v.ExpiredAt,
				CreatedAt: v.CreatedAt.UnixMilli(),
			})
		}
		return nil
	})
	if err := errWg.Wait(); err != nil {
		return err
	}
	apiResp.ApiRespOK(resp)
	return nil
}
