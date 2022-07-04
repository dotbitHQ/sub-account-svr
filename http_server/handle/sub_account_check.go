package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
)

type CheckSubAccount struct {
	CreateSubAccount
	Status  CheckStatus `json:"status"`
	Message string      `json:"message"`
}

type CheckStatus int

const (
	CheckStatusOk          CheckStatus = 0
	CheckStatusFail        CheckStatus = 1
	CheckStatusRegistered  CheckStatus = 2
	CheckStatusRegistering CheckStatus = 3
)

type RespSubAccountCheck struct {
	Result []CheckSubAccount `json:"result"`
}

func (h *HttpHandle) SubAccountCheck(ctx *gin.Context) {
	var (
		funcName = "SubAccountCheck"
		clientIp = GetClientIp(ctx)
		req      ReqSubAccountCreate
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

	if err = h.doSubAccountCheck(&req, &apiResp); err != nil {
		log.Error("doSubAccountCheck err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountCheck(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	// check params
	if err := h.doSubAccountCheckParams(req, apiResp); err != nil {
		return fmt.Errorf("doSubAccountCheckParams err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check account
	_, err := h.doSubAccountCheckAccount(req.Account, apiResp, common.DasActionCreateSubAccount)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckAccount err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check list
	_, resp, err := h.doSubAccountCheckList(req, apiResp)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckList err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doSubAccountCheckParams(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	if lenList := len(req.SubAccountList); lenList == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: len(sub account list) is 0")
		return nil
	} else if lenList > config.Cfg.Das.MaxCreateCount {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("more than max register num %d", config.Cfg.Das.MaxCreateCount))
		return nil
	}
	addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
		return nil
	}
	req.chainType, req.address = addrHex.ChainType, addrHex.AddressHex
	return nil
}

func (h *HttpHandle) doSubAccountCheckAccount(account string, apiResp *api_code.ApiResp, action common.DasAction) (*tables.TableAccountInfo, error) {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account")
		return nil, fmt.Errorf("GetAccountInfoByAccountId: %s", err.Error())
	} else if acc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return nil, nil
	} else if acc.Status != tables.AccountStatusNormal {
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "account on sale or auction")
		return nil, nil
	} else if acc.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, "account is expired")
		return nil, nil
	}
	switch action {
	case common.DasActionCreateSubAccount, common.DasActionEditSubAccount:
		if acc.EnableSubAccount != tables.AccountEnableStatusOn {
			apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOff, "sub account uninitialized")
			return nil, nil
		}
	}
	return &acc, nil
}

func (h *HttpHandle) doSubAccountCheckList(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) (bool, *RespSubAccountCheck, error) {
	isOk := true
	var resp RespSubAccountCheck
	resp.Result = make([]CheckSubAccount, 0)

	var subAccountMap = make(map[string]int)
	configCellBuilder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "failed to get config cell account")
		return false, nil, fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	maxLength, _ := configCellBuilder.MaxLength()
	// check list
	var accountIds []string
	for i, v := range req.SubAccountList {
		tmp := CheckSubAccount{
			CreateSubAccount: req.SubAccountList[i],
			Status:           0,
			Message:          "",
		}
		index := strings.Index(v.Account, ".")
		if index == -1 {
			tmp.Status = CheckStatusFail
			tmp.Message = fmt.Sprintf("sub account invalid: %s", v.Account)
			isOk = false
		} else {
			suffix := strings.TrimLeft(v.Account[index:], ".")
			if suffix != req.Account {
				tmp.Status = CheckStatusFail
				tmp.Message = fmt.Sprintf("account suffix diff: %s", suffix)
				isOk = false
			} else {
				accountId := common.Bytes2Hex(common.GetAccountIdByAccount(v.Account))
				accLen := common.GetAccountLength(v.Account[:index])
				if uint32(accLen) > maxLength {
					tmp.Status = CheckStatusFail
					tmp.Message = fmt.Sprintf("account len more than: %d", maxLength)
					isOk = false
				} else if index, ok := subAccountMap[accountId]; ok {
					resp.Result[index].Status = CheckStatusFail
					resp.Result[index].Message = fmt.Sprintf("same account")
					tmp.Status = CheckStatusFail
					tmp.Message = fmt.Sprintf("same account")
					isOk = false
				} else if v.RegisterYears <= 0 {
					tmp.Status = CheckStatusFail
					tmp.Message = "register years less than 1"
					isOk = false
				} else if v.RegisterYears > config.Cfg.Das.MaxRegisterYears {
					tmp.Status = CheckStatusFail
					tmp.Message = fmt.Sprintf("register years more than %d", config.Cfg.Das.MaxRegisterYears)
					isOk = false
				} else if _, err := common.AccountToAccountChars(v.Account[:strings.Index(v.Account, ".")]); err != nil {
					// check char set
					tmp.Status = CheckStatusFail
					tmp.Message = fmt.Sprintf("invalid character")
					isOk = false
				} else {
					addrHex, e := v.FormatChainTypeAddress(config.Cfg.Server.Net)
					if e != nil {
						tmp.Status = CheckStatusFail
						tmp.Message = fmt.Sprintf("params is invalid: %s", e.Error())
						isOk = false
					} else {
						accId := common.Bytes2Hex(common.GetAccountIdByAccount(v.Account))
						accountIds = append(accountIds, accId)
					}
					req.SubAccountList[i].chainType, req.SubAccountList[i].address = addrHex.ChainType, addrHex.AddressHex
				}
				subAccountMap[accountId] = i
			}
		}

		resp.Result = append(resp.Result, tmp)
	}

	// check registered
	registeredList, err := h.DbDao.GetAccountListByAccountIds(accountIds)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account list")
		return false, nil, fmt.Errorf("GetAccountListByAccountIds: %s", err.Error())
	}
	var mapRegistered = make(map[string]struct{})
	for _, v := range registeredList {
		mapRegistered[v.Account] = struct{}{}
	}

	// check registering
	registeringList, err := h.DbDao.GetSelfSmtRecordListByAccountIds(accountIds)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query smt record list")
		return false, nil, fmt.Errorf("GetSelfSmtRecordListByAccountIds: %s", err.Error())
	}
	var mapRegistering = make(map[string]struct{})
	for _, v := range registeringList {
		mapRegistering[v.Account] = struct{}{}
	}

	// check
	for i, v := range req.SubAccountList {
		if _, ok := mapRegistered[v.Account]; ok {
			isOk = false
			resp.Result[i].Status = CheckStatusRegistered
			resp.Result[i].Message = "registered"
		} else if _, ok = mapRegistering[v.Account]; ok {
			isOk = false
			resp.Result[i].Status = CheckStatusRegistering
			resp.Result[i].Message = "registering"
		}
	}
	return isOk, &resp, nil
}
