package handle

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"math"
	"net/http"
	"strings"
)

type ReqDistributionList struct {
	Account string `json:"account" binding:"required"`
	Page    int    `json:"page" binding:"gte=1"`
	Size    int    `json:"size" binding:"gte=1,lte=50"`
}

type RespDistributionList struct {
	Page  int                       `json:"page"`
	Total int64                     `json:"total"`
	List  []DistributionListElement `json:"list"`
}

type DistributionListElement struct {
	Time    int64  `json:"time"`
	Account string `json:"account"`
	Years   uint64 `json:"years"`
	Amount  string `json:"amount"`
	Symbol  string `json:"symbol"`
}

func (h *HttpHandle) DistributionList(ctx *gin.Context) {
	var (
		funcName               = "DistributionList"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqDistributionList
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

	if err = h.doDistributionList(&req, &apiResp); err != nil {
		log.Error("doDistributionList err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doDistributionList(req *ReqDistributionList, apiResp *api_code.ApiResp) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	if err := h.checkForSearch(accountId, apiResp); err != nil {
		return err
	}

	actions := []string{common.DasActionUpdateSubAccount, common.DasActionRenewSubAccount}
	subActions := []string{common.SubActionCreate, common.SubActionRenew}
	recordInfo, total, err := h.DbDao.FindSmtRecordInfoByActions(accountId, actions, subActions, req.Page, req.Size)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}

	resp := &RespDistributionList{
		Page:  req.Page,
		Total: total,
		List:  make([]DistributionListElement, len(recordInfo)),
	}
	if total == 0 {
		apiResp.ApiRespOK(resp)
		return nil
	}

	tokens, err := h.DbDao.FindTokens()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}

	ch := make(chan int, 10)
	errG := errgroup.Group{}
	errG.Go(func() error {
		for idx := range recordInfo {
			ch <- idx
		}
		close(ch)
		return nil
	})
	errG.Go(func() error {
		for v := range ch {
			idx := v
			errG.Go(func() error {
				record := recordInfo[idx]
				resp.List[idx] = DistributionListElement{
					Time:    record.CreatedAt.UnixMilli(),
					Account: strings.Split(record.Account, ".")[0],
					Years:   record.RegisterYears + record.RenewYears,
				}

				switch record.MintType {
				case tables.MintTypeDefault, tables.MintTypeManual:
					resp.List[idx].Amount = "0"
					resp.List[idx].Symbol = "Free mint by manager"
					return nil
				case tables.MintTypeAutoMint:
					order, err := h.DbDao.GetOrderByOrderID(record.OrderID)
					if err != nil {
						apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
						return err
					}
					if order.Id == 0 {
						err = fmt.Errorf("order: %s no exist", record.OrderID)
						apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, err.Error())
						return err
					}
					log.Infof("account: %s %d", order.Account, order.Amount.IntPart())
					token := tokens[order.TokenId]
					amount := order.Amount.Div(decimal.NewFromInt(int64(math.Pow10(int(token.Decimals)))))
					resp.List[idx].Amount = amount.String()
					resp.List[idx].Symbol = token.Symbol
				}
				return nil
			})
		}
		return nil
	})
	if err := errG.Wait(); err != nil {
		return err
	}
	apiResp.ApiRespOK(resp)
	return nil
}
