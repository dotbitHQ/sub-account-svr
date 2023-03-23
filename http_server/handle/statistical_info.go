package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/scorpiotzh/toolib"
	"golang.org/x/sync/errgroup"
	"net/http"
	"strings"
)

type ReqStatisticalInfo struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
}

type RespStatisticalInfo struct {
	SubAccountNum int64        `json:"sub_account_num"`
	AddressNum    int64        `json:"address_num"`
	IncomeInfo    []IncomeInfo `json:"income_info"`
	CkbSpending   CkbSpending  `json:"ckb_spending"`
}

type IncomeInfo struct {
	Type    string `json:"type"`
	Balance string `json:"balance"`
	Total   string `json:"total"`
}

type CkbSpending struct {
	Balance string `json:"balance"`
	Total   string `json:"total"`
}

func (h *HttpHandle) StatisticalInfo(ctx *gin.Context) {
	var (
		funcName = "StatisticalInfo"
		clientIp = GetClientIp(ctx)
		req      ReqStatisticalInfo
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

	if err = h.doStatisticalInfo(&req, &apiResp); err != nil {
		log.Error("doStatisticalInfo err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doStatisticalInfo(req *ReqStatisticalInfo, apiResp *api_code.ApiResp) error {
	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return err
	}
	address := res.AddressHex
	if strings.HasPrefix(res.AddressHex, common.HexPreFix) {
		address = strings.ToLower(res.AddressHex)
	}
	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "search account err")
		return fmt.Errorf("SearchAccount err: %s", err.Error())
	}

	resp := RespStatisticalInfo{
		IncomeInfo: []IncomeInfo{},
	}
	errG := &errgroup.Group{}

	errG.Go(func() error {
		subAccountNum, err := h.DbDao.GetSubAccountNum(req.Account)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
			return err
		}
		resp.SubAccountNum = subAccountNum
		return nil
	})

	errG.Go(func() error {
		subAccountDistinct, err := h.DbDao.GetSubAccountNumDistinct(req.Account)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
			return err
		}
		resp.AddressNum = subAccountDistinct
		return nil
	})

	errG.Go(func() error {
		smtRecords, err := h.DbDao.FindSmtRecordInfoByMintType(accountId, tables.MintTypeAutoMint, []string{common.DasActionCreateSubAccount, common.DasActionRenewSubAccount})
		if err != nil {
			return err
		}

		type Income struct {
			Type    string
			Total   float64
			Balance float64
		}
		paymentInfo := make(map[string]*Income)

		for _, record := range smtRecords {
			paymentsInfo, err := h.DbDao.FindPaymentInfoByOrderId(record.OrderID)
			if err != nil {
				return err
			}
			for _, payment := range paymentsInfo {
				token, err := h.DbDao.GetTokenById(payment.TokenId)
				if err != nil {
					return err
				}
				p, ok := paymentInfo[token.TokenId]
				if !ok {
					p = &Income{
						Type: token.Symbol,
					}
					paymentInfo[token.TokenId] = p
				}
				p.Total += payment.Amount
			}
		}

		for k, v := range paymentInfo {
			amount, err := h.DbDao.GetAutoPaymentAmount(accountId, k, tables.PaymentStatusSuccess)
			if err != nil {
				return err
			}
			v.Balance += amount
		}

		for _, v := range paymentInfo {
			resp.IncomeInfo = append(resp.IncomeInfo, IncomeInfo{
				Type:    v.Type,
				Total:   fmt.Sprintf("%f", v.Total),
				Balance: fmt.Sprintf("%f", v.Total-v.Balance),
			})
		}
		return nil
	})

	errG.Go(func() error {
		daf := core.DasAddressFormat{DasNetType: config.Cfg.Server.Net}
		addrHex, err := daf.NormalToHex(core.DasAddressNormal{
			ChainType:     acc.OwnerChainType,
			AddressNormal: acc.Owner,
			Is712:         true,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("address NormalToHex err: %s", err.Error())
		}
		dasLock, _, err := h.DasCore.Daf().HexToScript(addrHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("HexToScript err: %s", err.Error())
		}
		_, totalCapacity, err := h.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
			DasCache:          h.DasCache,
			LockScript:        dasLock,
			CapacityForChange: common.DasLockWithBalanceTypeOccupiedCkb,
			SearchOrder:       indexer.SearchOrderDesc,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("GetBalanceCells err: %s", err)
		}
		resp.CkbSpending.Balance = fmt.Sprintf("%d", totalCapacity)
		return nil
	})

	errG.Go(func() error {
		total, err := h.DbDao.GetSmtRecordManualMintYears(accountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
			return err
		}
		resp.CkbSpending.Total = fmt.Sprintf("%d", total)
		return nil
	})

	if err := errG.Wait(); err != nil {
		return err
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) checkAuth(address, account string, apiResp *api_code.ApiResp) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "search account err")
		return fmt.Errorf("SearchAccount err: %s", err.Error())
	}
	if acc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return fmt.Errorf("account not exist: %s", account)
	}

	if !strings.EqualFold(acc.Owner, address) && !strings.EqualFold(acc.Manager, address) {
		err = errors.New("you not this account permissions")
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
		return err
	}
	return nil
}
