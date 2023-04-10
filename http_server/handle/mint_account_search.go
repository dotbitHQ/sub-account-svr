package handle

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"strings"
)

type ReqMintAccountSearch struct {
	core.ChainTypeAddress
	SubAccount string `json:"sub_account"`
}

type RespMintAccountSearch struct {
	Price   decimal.Decimal `json:"price"`
	MaxYear int64           `json:"max_year"`
	Status  AccStatus       `json:"status"`
	IsSelf  bool            `json:"is_self"`
}

type AccStatus int

const (
	AccStatusUnregistered AccStatus = 0
	AccStatusRegistering  AccStatus = 1
	AccStatusRegistered   AccStatus = 2
)

func (h *HttpHandle) MintAccountSearch(ctx *gin.Context) {
	var (
		funcName = "MintAccountSearch"
		clientIp = GetClientIp(ctx)
		req      ReqMintAccountSearch
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

	if err = h.doMintAccountSearch(&req, &apiResp); err != nil {
		log.Error("doMintAccountSearch err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doMintAccountSearch(req *ReqMintAccountSearch, apiResp *api_code.ApiResp) error {
	var resp RespMintAccountSearch
	// check key info
	hexAddr, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("key-info[%s-%s] invalid", req.KeyInfo.CoinType, req.KeyInfo.Key))
		return nil
	}
	// check sub account name
	if !strings.HasSuffix(req.SubAccount, common.DasAccountSuffix) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("sub-account[%s] invalid", req.SubAccount))
		return nil
	}
	if strings.Count(req.SubAccount, ".") != 2 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("sub-account[%s] invalid", req.SubAccount))
		return nil
	}
	indexDot := strings.Index(req.SubAccount, ".")
	if indexDot == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("sub-account[%s] invalid", req.SubAccount))
		return nil
	}
	// check parent account
	parentAccountName := req.SubAccount[indexDot+1:]
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(parentAccountName))
	parentAccount, err := h.DbDao.GetAccountInfoByAccountId(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s %s", err.Error(), parentAccountId)
	} else if parentAccount.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOff, "The sub-account function is not activated")
		return nil
	} else if parentAccount.Status != tables.AccountStatusNormal {
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "account status is not normal")
		return nil
	} else if parentAccount.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, "account is expired")
		return nil
	}
	// check sub account
	subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccount))
	subAccount, err := h.DbDao.GetAccountInfoByAccountId(subAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query sub-account")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s %s", err.Error(), subAccountId)
	} else if subAccount.Id > 0 {
		resp.Status = AccStatusUnregistered
	} else {
		// check order
		orderInfo, err := h.DbDao.GetMintOrderInProgressByAccountIdWithAddr(subAccountId, hexAddr.AddressHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order")
			return fmt.Errorf("GetMintOrderInProgressByAccountIdWithAddr err: %s %s", err.Error(), subAccountId)
		} else if orderInfo.Id > 0 {
			resp.Status = AccStatusRegistering
			resp.IsSelf = true
		} else {
			orderInfo, err = h.DbDao.GetMintOrderInProgressByAccountIdWithoutAddr(subAccountId, hexAddr.AddressHex)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order")
				return fmt.Errorf("GetMintOrderInProgressByAccountIdWithAddr err: %s %s", err.Error(), subAccountId)
			} else if orderInfo.Id > 0 {
				resp.Status = AccStatusRegistering
			}
		}
	}
	// todo check price: blacklist or price rule

	// check max years
	resp.MaxYear = int64(parentAccount.ExpiredAt) / common.OneYearSec
	if resp.MaxYear == 0 {
		resp.MaxYear = 1
	}

	// todo get price
	resp.Price = decimal.Zero

	apiResp.ApiRespOK(resp)
	return nil
}
