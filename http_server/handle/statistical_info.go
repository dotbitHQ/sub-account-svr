package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"math"
	"net/http"
	"sort"
	"strings"
)

type ReqStatisticalInfo struct {
	Account string `json:"account" binding:"required"`
}

type RespStatisticalInfo struct {
	SubAccountNum int64        `json:"sub_account_num"`
	AddressNum    int64        `json:"address_num"`
	IncomeInfo    []IncomeInfo `json:"income_info"`
	CkbSpending   CkbSpending  `json:"ckb_spending"`
	AutoMint      struct {
		Enable          bool  `json:"enable"`
		FirstEnableTime int64 `json:"first_enable_time"`
	} `json:"auto_mint"`
	AccountExpiredAt uint64 `json:"account_expired_at"`
}

type IncomeInfo struct {
	Type            string `json:"type"`
	Balance         string `json:"balance"`
	Total           string `json:"total"`
	BackgroundColor string `json:"background_color"`
}

type CkbSpending struct {
	Balance string `json:"balance"`
	Total   string `json:"total"`
}

func (h *HttpHandle) StatisticalInfo(ctx *gin.Context) {
	var (
		funcName               = "StatisticalInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqStatisticalInfo
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

	if err = h.doStatisticalInfo(&req, &apiResp); err != nil {
		log.Error("doStatisticalInfo err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doStatisticalInfo(req *ReqStatisticalInfo, apiResp *api_code.ApiResp) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	if err := h.checkForSearch(accountId, apiResp); err != nil {
		return err
	}

	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "search account err")
		return fmt.Errorf("SearchAccount err: %s", err.Error())
	}

	resp := RespStatisticalInfo{
		IncomeInfo:       []IncomeInfo{},
		AccountExpiredAt: acc.ExpiredAt * 1e3,
	}
	errG := &errgroup.Group{}

	errG.Go(func() error {
		subAccountNum, err := h.DbDao.GetSubAccountNum(accountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
			return err
		}
		resp.SubAccountNum = subAccountNum
		return nil
	})

	errG.Go(func() error {
		subAccountDistinct, err := h.DbDao.GetSubAccountNumDistinct(accountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
			return err
		}
		resp.AddressNum = subAccountDistinct
		return nil
	})

	errG.Go(func() error {
		totalAmount, err := h.DbDao.GetOrderAmount(accountId, false)
		if err != nil {
			return err
		}
		paidAmount, err := h.DbDao.GetOrderAmount(accountId, true)
		if err != nil {
			return err
		}

		paymentConfig, err := h.DbDao.GetUserPaymentConfig(accountId)
		if err != nil {
			return err
		}
		for k, v := range paymentConfig.CfgMap {
			if _, ok := totalAmount[k]; !ok && v.Enable {
				totalAmount[k] = decimal.NewFromInt(0)
			}
		}

		tokens, err := h.DbDao.FindTokens()
		if err != nil {
			return err
		}

		for k, v := range totalAmount {
			token := tokens[k]
			if v.Sub(paidAmount[k]).LessThanOrEqual(decimal.NewFromInt(0)) &&
				!paymentConfig.CfgMap[k].Enable {
				continue
			}
			decimals := decimal.NewFromInt(int64(math.Pow10(int(token.Decimals))))
			rate := decimal.NewFromFloat(1 - config.Cfg.Das.AutoMint.ServiceFeeRatio)
			total := v.Mul(rate).DivRound(decimals, token.Decimals)
			balance := v.Sub(paidAmount[k]).Mul(rate).DivRound(decimals, token.Decimals)

			resp.IncomeInfo = append(resp.IncomeInfo, IncomeInfo{
				Type:            token.Symbol,
				Total:           total.String(),
				Balance:         balance.String(),
				BackgroundColor: config.Cfg.Das.AutoMint.BackgroundColors[k],
			})
		}

		sort.Slice(resp.IncomeInfo, func(i, j int) bool {
			return resp.IncomeInfo[i].Type < resp.IncomeInfo[j].Type
		})
		return nil
	})

	errG.Go(func() error {
		dasLock, _, err := h.DasCore.Daf().HexToScript(core.DasAddressHex{
			DasAlgorithmId: acc.ManagerAlgorithmId,
			AddressHex:     acc.Manager,
			ChainType:      acc.ManagerChainType,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("HexToScript err: %s", err.Error())
		}
		_, totalCapacity, err := h.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
			LockScript:  dasLock,
			SearchOrder: indexer.SearchOrderDesc,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("GetBalanceCells err: %s", err)
		}

		log.Infof("totalCapacity: %d", totalCapacity)

		token, err := h.DbDao.GetTokenById(tables.TokenIdCkb)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
			return fmt.Errorf("GetTokenById err: %s", err)
		}
		decimals := decimal.NewFromInt(int64(math.Pow10(int(token.Decimals))))
		resp.CkbSpending.Balance = decimal.NewFromInt(int64(totalCapacity)).Div(decimals).String()
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

	errG.Go(func() error {
		baseInfo, err := h.TxTool.GetBaseInfo()
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
			return err
		}
		subAccountCell, err := h.getSubAccountCell(baseInfo.ContractSubAcc, accountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
			return fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
		}
		subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
			return err

		}
		subAccountCellDetail := witness.ConvertSubAccountCellOutputData(subAccountTx.Transaction.OutputsData[subAccountCell.OutPoint.Index])

		log.Info("doStatisticalInfo:", subAccountCell.OutPoint.TxHash.String(), subAccountCell.OutPoint.Index)

		if subAccountCellDetail.Flag == witness.FlagTypeCustomRule &&
			subAccountCellDetail.AutoDistribution == witness.AutoDistributionEnable {
			resp.AutoMint.Enable = true
		}

		first, err := h.DbDao.FirstEnableAutoMint(accountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
			return err
		}
		if first.Id > 0 {
			resp.AutoMint.FirstEnableTime = first.Timestamp
		}
		return nil
	})

	if err := errG.Wait(); err != nil {
		return err
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) check(address, account string, action common.DasAction, apiResp *api_code.ApiResp) error {
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

	accountInfo, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "search account err")
		return fmt.Errorf("SearchAccount err: %s", err.Error())
	}
	if accountInfo.EnableSubAccount != tables.AccountEnableStatusOn {
		err = errors.New("sub account no enable, please enable sub_account before use")
		apiResp.ApiRespErr(api_code.ApiCodeSubAccountNoEnable, err.Error())
		return err
	}
	if accountInfo.IsExpired() {
		err = errors.New("account expired, please renew before use")
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountExpired, err.Error())
		return err
	}

	if action == common.DasActionConfigSubAccount {
		task, err := h.DbDao.GetLatestTask(accountId, common.DasActionConfigSubAccount)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
			return err
		}
		if task.Id > 0 && task.TxStatus == tables.TxStatusPending {
			err = errors.New("sub account pending, please wait")
			apiResp.ApiRespErr(api_code.ApiCodeConfigSubAccountPending, err.Error())
			return err
		}
	}
	return nil
}
