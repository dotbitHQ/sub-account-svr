package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"golang.org/x/sync/errgroup"
	"net/http"
	"regexp"
	"strings"
)

type ReqStatisticalInfo struct {
	core.ChainTypeAddress
	Account string `json:"account"`
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
		log.Error("doSubAccountMintStatus err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doStatisticalInfo(req *ReqStatisticalInfo, apiResp *api_code.ApiResp) error {
	res := checkReqKeyInfo(h.DasCore.Daf(), &req.ChainTypeAddress, apiResp)
	if res == nil {
		return nil
	}
	address := strings.ToLower(res.AddressHex)
	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}

	resp := RespStatisticalInfo{}
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

	// TODO

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

// checkReqKeyInfo
func checkReqKeyInfo(daf *core.DasAddressFormat, req *core.ChainTypeAddress, apiResp *api_code.ApiResp) *core.DasAddressHex {
	if req.Type != "blockchain" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("type [%s] is invalid", req.Type))
		return nil
	}
	if req.KeyInfo.Key == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "key is invalid")
		return nil
	}
	dasChainType := common.FormatCoinTypeToDasChainType(req.KeyInfo.CoinType)
	if dasChainType == -1 {
		dasChainType = common.FormatChainIdToDasChainType(config.Cfg.Server.Net, req.KeyInfo.ChainId)
	}
	if dasChainType == -1 {
		if !strings.HasPrefix(req.KeyInfo.Key, "0x") {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("coin_type [%s] and chain_id [%s] is invalid", req.KeyInfo.CoinType, req.KeyInfo.ChainId))
			return nil
		}

		ok, err := regexp.MatchString("^0x[0-9a-fA-F]{40}$", req.KeyInfo.Key)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
			return nil
		}

		if ok {
			dasChainType = common.ChainTypeEth
		} else {
			ok, err = regexp.MatchString("^0x[0-9a-fA-F]{64}$", req.KeyInfo.Key)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
				return nil
			}
			if !ok {
				apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "key is invalid")
				return nil
			}
			dasChainType = common.ChainTypeMixin
		}
	}
	addrHex, err := daf.NormalToHex(core.DasAddressNormal{
		ChainType:     dasChainType,
		AddressNormal: req.KeyInfo.Key,
		Is712:         true,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
		return nil
	}
	if addrHex.DasAlgorithmId == common.DasAlgorithmIdEth712 {
		addrHex.DasAlgorithmId = common.DasAlgorithmIdEth
	}
	return &addrHex
}
