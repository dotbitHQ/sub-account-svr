package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"golang.org/x/sync/errgroup"
	"math"
	"net/http"
)

type ReqDistributionList struct {
	core.ChainTypeAddress
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
	Time    string `json:"time"`
	Account string `json:"account"`
	Year    uint64 `json:"year"`
	Amount  string `json:"amount"`
}

func (h *HttpHandle) DistributionList(ctx *gin.Context) {
	var (
		funcName = "DistributionList"
		clientIp = GetClientIp(ctx)
		req      ReqDistributionList
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doDistributionList(&req, &apiResp); err != nil {
		log.Error("doDistributionList err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doDistributionList(req *ReqDistributionList, apiResp *api_code.ApiResp) error {
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}
	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return err
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)
	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	recordInfo, total, err := h.DbDao.FindSmtRecordInfoByMintTypeAndPaging(accountId, []tables.MintType{tables.MintTypeDefault, tables.MintTypeManual, tables.MintTypeAutoMint}, []string{common.DasActionCreateSubAccount, common.DasActionRenewSubAccount}, req.Page, req.Size)
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
					Time:    record.CreatedAt.Format("2006-01-02 15:04"),
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
					paymentInfo, err := h.DbDao.GetPaymentInfoByOrderId(record.OrderID)
					if err != nil {
						apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
						return errors.New("db error")
					}
					if paymentInfo.Id == 0 {
						log.Warnf("order: %s no payment info", record.OrderID)
						return nil
					}
					token, err := h.DbDao.GetTokenById(order.TokenId)
					if err != nil {
						apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
						return errors.New("db error")
					}
					list[idx].Amount = fmt.Sprintf("%f %s", order.Amount.InexactFloat64()/math.Pow10(token.Decimals), token.Symbol)
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
