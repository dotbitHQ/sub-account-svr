package handle

import (
	"das_sub_account/http_server/api_code"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"math"
	"net/http"
)

type ReqAutoPaymentList struct {
	Account string `json:"account" binding:"required"`
	Page    int    `json:"page" binding:"required,min=1"`
	Size    int    `json:"size" binding:"required,min=1,max=100"`
}

type RespAutoPaymentList struct {
	Total int64             `json:"total"`
	List  []AutoPaymentData `json:"list"`
}

type AutoPaymentData struct {
	Time   int64  `json:"time"`
	Amount string `json:"amount"`
}

func (h *HttpHandle) AutoPaymentList(ctx *gin.Context) {
	var (
		funcName               = "AutoPaymentList"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqAutoPaymentList
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

	if err = h.autoPaymentList(&req, &apiResp); err != nil {
		log.Error("autoPaymentList err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) autoPaymentList(req *ReqAutoPaymentList, apiResp *api_code.ApiResp) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	if err := h.checkForSearch(accountId, apiResp); err != nil {
		return err
	}

	res, total, err := h.DbDao.FindAutoPaymentInfo(accountId, req.Page, req.Size)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return err
	}
	resp := &RespAutoPaymentList{
		Total: total,
		List:  make([]AutoPaymentData, 0),
	}

	for _, v := range res {
		token, err := h.DbDao.GetTokenById(v.TokenId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return err
		}
		amount := v.Amount.DivRound(decimal.NewFromInt(int64(math.Pow10(int(token.Decimals)))), token.Decimals)
		resp.List = append(resp.List, AutoPaymentData{
			Time:   v.CreatedAt.UnixMilli(),
			Amount: fmt.Sprintf("%s %s", amount.String(), token.Symbol),
		})
	}

	apiResp.ApiRespOK(resp)
	return nil
}
