package handle

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"math"
	"net/http"
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
	Year    uint64 `json:"year"`
	Amount  string `json:"amount"`
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

	recordInfo, total, err := h.DbDao.FindSmtRecordInfoByActions(accountId, []string{common.DasActionUpdateSubAccount, common.DasActionRenewSubAccount}, req.Page, req.Size)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}
	resp := &RespDistributionList{
		Page:  req.Page,
		Total: total,
	}

	ch := make(chan int, 10)
	list := make([]DistributionListElement, len(recordInfo))

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
				list[idx] = DistributionListElement{
					Time:    record.CreatedAt.UnixMilli(),
					Account: record.Account,
					Year:    record.RegisterYears + record.RenewYears,
				}

				switch record.MintType {
				case tables.MintTypeDefault, tables.MintTypeManual:
					list[idx].Amount = "Free mint by manager"
					return nil
				case tables.MintTypeAutoMint:
					order, err := h.DbDao.GetOrderByOrderID(record.OrderID)
					if err != nil {
						apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
						return err
					}
					if order.Id == 0 {
						apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
						return errors.New("db error")
					}

					log.Infof("account: %s %d", order.Account, order.Amount.IntPart())

					token, err := h.DbDao.GetTokenById(order.TokenId)
					if err != nil {
						apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
						return errors.New("db error")
					}
					amount := order.Amount.Div(decimal.NewFromInt(int64(math.Pow10(int(token.Decimals)))))
					list[idx].Amount = fmt.Sprintf("%s %s", amount, token.Symbol)
				default:
					apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
					return errors.New("db error")
				}
				return nil
			})
		}
		return nil
	})
	if err := errG.Wait(); err != nil {
		return err
	}
	resp.List = list
	apiResp.ApiRespOK(resp)
	return nil
}
