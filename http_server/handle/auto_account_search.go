package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"strings"
	"time"
)

type ReqAutoAccountSearch struct {
	core.ChainTypeAddress
	SubAccount string `json:"sub_account"`
}

type RespAutoAccountSearch struct {
	Price   decimal.Decimal `json:"price"`
	MaxYear uint64          `json:"max_year"`
	Status  AccStatus       `json:"status"`
	IsSelf  bool            `json:"is_self"`
}

type AccStatus int

const (
	AccStatusUnMinted AccStatus = 0
	AccStatusMinting  AccStatus = 1
	AccStatusMinted             = 2
)

func (h *HttpHandle) AutoAccountSearch(ctx *gin.Context) {
	var (
		funcName               = "AutoAccountSearch"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqAutoAccountSearch
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

	if err = h.doAutoAccountSearch(&req, &apiResp); err != nil {
		log.Error("doAutoAccountSearch err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoAccountSearch(req *ReqAutoAccountSearch, apiResp *api_code.ApiResp) error {
	var resp RespAutoAccountSearch
	// check key info
	hexAddr, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("key-info[%s-%s] invalid", req.KeyInfo.CoinType, req.KeyInfo.Key))
		return nil
	}

	// check sub account name
	parentAccountId := h.checkSubAccountName(apiResp, req.SubAccount)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check parent account
	parentAccount, err := h.checkParentAccount(apiResp, parentAccountId)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check sub account
	subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccount))
	_, resp.Status, resp.IsSelf, err = h.checkSubAccount(apiResp, hexAddr, subAccountId)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// get max years
	resp.MaxYear = h.getMaxYears(parentAccount)

	// get rule price
	resp.Price, err = h.getRulePrice(parentAccountId, req.SubAccount, apiResp)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) checkSubAccountName(apiResp *api_code.ApiResp, subAccountName string) (parentAccountId string) {
	if !strings.HasSuffix(subAccountName, common.DasAccountSuffix) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("sub-account[%s] invalid", subAccountName))
		return
	}
	if strings.Count(subAccountName, ".") != 2 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("sub-account[%s] invalid", subAccountName))
		return
	}
	indexDot := strings.Index(subAccountName, ".")
	if indexDot == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("sub-account[%s] invalid", subAccountName))
		return
	}
	parentAccountName := subAccountName[indexDot+1:]
	parentAccountId = common.Bytes2Hex(common.GetAccountIdByAccount(parentAccountName))
	return
}

func (h *HttpHandle) checkParentAccount(apiResp *api_code.ApiResp, parentAccountId string) (*tables.TableAccountInfo, error) {
	parentAccount, err := h.DbDao.GetAccountInfoByAccountId(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		return nil, fmt.Errorf("GetAccountInfoByAccountId err: %s %s", err.Error(), parentAccountId)
	} else if parentAccount.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountNotExist, "parent account does not exist")
		return nil, nil
	} else if parentAccount.Status != tables.AccountStatusNormal {
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "parent account status is not normal")
		return nil, nil
	} else if parentAccount.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, "parent account is expired")
		return nil, nil
	}
	return &parentAccount, nil
}

func (h *HttpHandle) checkSubAccount(apiResp *api_code.ApiResp, hexAddr *core.DasAddressHex, subAccountId string) (orderInfo tables.OrderInfo, accStatus AccStatus, isSelf bool, e error) {
	accStatus = AccStatusUnMinted
	subAccount, err := h.DbDao.GetAccountInfoByAccountId(subAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query sub-account")
		e = fmt.Errorf("GetAccountInfoByAccountId err: %s %s", err.Error(), subAccountId)
		return
	} else if subAccount.Id > 0 {
		accStatus = AccStatusMinted
		return
	}
	// check order of self
	orderInfo, err = h.DbDao.GetMintOrderInProgressByAccountIdWithAddr(subAccountId, hexAddr.AddressHex)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order")
		e = fmt.Errorf("GetMintOrderInProgressByAccountIdWithAddr err: %s %s", err.Error(), subAccountId)
		return
	} else if orderInfo.Id > 0 {
		isSelf, accStatus = true, AccStatusMinting
		return
	}
	// check order of others
	orderInfo, err = h.DbDao.GetMintOrderInProgressByAccountIdWithoutAddr(subAccountId, hexAddr.AddressHex)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order")
		e = fmt.Errorf("GetMintOrderInProgressByAccountIdWithAddr err: %s %s", err.Error(), subAccountId)
		return
	} else if orderInfo.Id > 0 {
		accStatus = AccStatusMinting
		return
	}
	return
}

func (h *HttpHandle) getMaxYears(parentAccount *tables.TableAccountInfo) uint64 {
	//1713261581
	nowT := uint64(time.Now().Unix())
	if nowT > parentAccount.ExpiredAt {
		return 0
	}

	maxYear := (parentAccount.ExpiredAt - nowT) / uint64(common.OneYearSec)
	log.Info("getMaxYears:", parentAccount.ExpiredAt, maxYear, config.Cfg.Das.MaxRegisterYears)
	if maxYear > config.Cfg.Das.MaxRegisterYears {
		maxYear = config.Cfg.Das.MaxRegisterYears
	}
	return maxYear
}

func (h *HttpHandle) getRulePrice(parentAccountId, subAccount string, apiResp *api_code.ApiResp) (price decimal.Decimal, e error) {
	ruleConfig, err := h.DbDao.GetRuleConfigByAccountId(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to search rule config")
		e = fmt.Errorf("GetRuleConfigByAccountId err: %s", err.Error())
		return
	}
	ruleTx, err := h.DasCore.Client().GetTransaction(h.Ctx, types.HexToHash(ruleConfig.TxHash))
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to search rule tx")
		e = fmt.Errorf("GetTransaction err: %s", err.Error())
		return
	}
	var ruleReverse witness.SubAccountRuleEntity
	if err = ruleReverse.ParseFromTx(ruleTx.Transaction, common.ActionDataTypeSubAccountPreservedRules); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to search rules")
		e = fmt.Errorf("ParseFromTx err: %s", err.Error())
		return
	}
	hit, index, err := ruleReverse.Hit(subAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to match rules")
		e = fmt.Errorf("ruleReverse.Hit err: %s", err.Error())
		return
	} else if hit {
		apiResp.ApiRespErr(api_code.ApiCodeHitBlacklist, "hit blacklist")
		return
	}

	var rulePrice witness.SubAccountRuleEntity
	if err = rulePrice.ParseFromTx(ruleTx.Transaction, common.ActionDataTypeSubAccountPriceRules); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to search rules")
		e = fmt.Errorf("ParseFromTx err: %s", err.Error())
		return
	}
	hit, index, err = rulePrice.Hit(subAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to match rules")
		e = fmt.Errorf("rulePrice.Hit err: %s", err.Error())
		return
	} else if !hit {
		apiResp.ApiRespErr(api_code.ApiCodeNoTSetRules, "not set price rules")
		return
	}
	price = decimal.NewFromInt(int64(rulePrice.Rules[index].Price))
	return
}
