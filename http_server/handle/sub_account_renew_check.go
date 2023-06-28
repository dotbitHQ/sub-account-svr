package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

type CheckSubAccountRenew struct {
	RenewSubAccount
	Status  RenewCheckStatus `json:"status"`
	Message string           `json:"message"`
}

type RenewCheckStatus int

const (
	RenewCheckStatusOk                    RenewCheckStatus = 0
	RenewCheckStatusFail                  RenewCheckStatus = 1
	RenewCheckStatusNoExist               RenewCheckStatus = 2
	RenewCheckStatusExpirationGracePeriod RenewCheckStatus = 3
	RenewCheckStatusExpired               RenewCheckStatus = 4
	RenewCheckStatusRegistering           RenewCheckStatus = 5
	RenewCheckStatusHitPreservedRules     RenewCheckStatus = 6
)

type RespSubAccountRenewCheck struct {
	Result []CheckSubAccountRenew `json:"result"`
}

func (h *HttpHandle) SubAccountRenewCheck(ctx *gin.Context) {
	var (
		funcName               = "SubAccountRenewCheck"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSubAccountRenew
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

	if err = h.doSubAccountRenewCheck(&req, &apiResp); err != nil {
		log.Error("doSubAccountCheck err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountRenewCheck(req *ReqSubAccountRenew, apiResp *api_code.ApiResp) error {
	// check params
	if err := h.doSubAccountRenewCheckParams(req, apiResp); err != nil {
		return fmt.Errorf("doSubAccountRenewCheckParams err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check account
	_, err := h.doSubAccountRenewCheckAccount(req, apiResp)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckAccount err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check list
	_, resp, err := h.doSubAccountRenewCheckList(req, apiResp)
	if err != nil {
		return fmt.Errorf("doSubAccountRenewCheckList err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doSubAccountRenewCheckList(req *ReqSubAccountRenew, apiResp *api_code.ApiResp) (bool, *RespSubAccountRenewCheck, error) {
	isOk := true
	var resp RespSubAccountRenewCheck
	resp.Result = make([]CheckSubAccountRenew, 0)

	var subAccountMap = make(map[string]int)
	configCellBuilder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "failed to get config cell account")
		return false, nil, fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	maxLength, _ := configCellBuilder.MaxLength()
	expirationGracePeriod, err := configCellBuilder.ExpirationGracePeriod()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return false, nil, err
	}

	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	baseInfo, err := h.TxTool.GetBaseInfo()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return false, nil, err
	}

	subAccountCell, err := h.getSubAccountCell(baseInfo.ContractSubAcc, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return false, nil, fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return false, nil, err
	}

	var subAccountEntity *witness.SubAccountRuleEntity
	subAccountCellDetail := witness.ConvertSubAccountCellOutputData(subAccountTx.Transaction.OutputsData[subAccountCell.OutPoint.Index])
	if subAccountCellDetail.Flag == witness.FlagTypeCustomRule {
		// check custom rule
		subAccountEntity = witness.NewSubAccountRuleEntity(req.Account)
		if err := subAccountEntity.ParseFromTx(subAccountTx.Transaction, common.ActionDataTypeSubAccountPreservedRules); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
			return false, nil, err
		}
	}

	// check list
	var accountIds []string
	for i := range req.SubAccountList {
		tmp := CheckSubAccountRenew{
			RenewSubAccount: req.SubAccountList[i],
		}
		index := strings.Index(req.SubAccountList[i].Account, ".")

		if index == -1 {
			tmp.Status = RenewCheckStatusFail
			tmp.Message = fmt.Sprintf("sub account invalid: %s", req.SubAccountList[i].Account)
			isOk = false
			resp.Result = append(resp.Result, tmp)
			continue
		}

		suffix := strings.TrimLeft(req.SubAccountList[i].Account[index:], ".")
		if suffix != req.Account {
			tmp.Status = RenewCheckStatusFail
			tmp.Message = fmt.Sprintf("account suffix diff: %s", suffix)
			isOk = false
			resp.Result = append(resp.Result, tmp)
			continue
		}
		accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccountList[i].Account))

		if len(req.SubAccountList[i].AccountCharStr) == 0 {
			accountCharStr, err := h.DasCore.GetAccountCharSetList(req.SubAccountList[i].Account)
			if err != nil {
				tmp.Status = RenewCheckStatusFail
				tmp.Message = fmt.Sprintf("AccountToAccountChars err: %s", suffix)
				isOk = false
				resp.Result = append(resp.Result, tmp)
				continue
			}
			req.SubAccountList[i].AccountCharStr = accountCharStr
		}

		accLen := len(req.SubAccountList[i].AccountCharStr)
		if accLen <= 0 {
			tmp.Status = RenewCheckStatusFail
			tmp.Message = fmt.Sprintf("account length is 0")
			isOk = false
		} else if uint32(accLen) > maxLength {
			tmp.Status = RenewCheckStatusFail
			tmp.Message = fmt.Sprintf("account len more than: %d", maxLength)
			isOk = false
		}

		if indexAcc, ok := subAccountMap[accountId]; ok {
			resp.Result[indexAcc].Status = RenewCheckStatusFail
			resp.Result[indexAcc].Message = fmt.Sprintf("same account")
			tmp.Status = RenewCheckStatusFail
			tmp.Message = fmt.Sprintf("same account")
			isOk = false
		} else if req.SubAccountList[i].RenewYears <= 0 {
			tmp.Status = RenewCheckStatusFail
			tmp.Message = "register years less than 1"
			isOk = false
		} else if req.SubAccountList[i].RenewYears > config.Cfg.Das.MaxRenewYears {
			tmp.Status = RenewCheckStatusFail
			tmp.Message = fmt.Sprintf("renew years more than %d", config.Cfg.Das.MaxRenewYears)
			isOk = false
		} else if !h.checkAccountCharSet(req.SubAccountList[i].AccountCharStr, req.SubAccountList[i].Account[:strings.Index(req.SubAccountList[i].Account, ".")]) {
			log.Info("checkAccountCharSet:", req.SubAccountList[i].Account, req.SubAccountList[i].AccountCharStr)
			tmp.Status = RenewCheckStatusFail
			tmp.Message = fmt.Sprintf("checkAccountCharSet invalid charset")
			isOk = false
		}

		if subAccountEntity != nil {
			hit, _, err := subAccountEntity.Hit(req.SubAccountList[i].Account)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
				return false, nil, err
			}
			if hit {
				tmp.Status = RenewCheckStatusHitPreservedRules
				tmp.Message = "sub account is preserved"
				isOk = false
			}
		}
		if tmp.Status != RenewCheckStatusOk {
			resp.Result = append(resp.Result, tmp)
			continue
		}

		accountIds = append(accountIds, accountId)
		subAccountMap[accountId] = i
		resp.Result = append(resp.Result, tmp)
	}

	// check registered
	registeredList, err := h.DbDao.GetAccountListByAccountIds(accountIds)
	if err != nil && err != gorm.ErrRecordNotFound {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account list")
		return false, nil, fmt.Errorf("GetAccountListByAccountIds: %s", err.Error())
	}
	var mapRegistered = make(map[string]*tables.TableAccountInfo)
	for i := range registeredList {
		mapRegistered[registeredList[i].Account] = &registeredList[i]
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
	now := time.Now().Unix()
	for i, v := range req.SubAccountList {
		acc, ok := mapRegistered[v.Account]
		if !ok {
			isOk = false
			if _, ok = mapRegistering[v.Account]; ok {
				resp.Result[i].Status = RenewCheckStatusRegistering
				resp.Result[i].Message = "registering"
			} else {
				resp.Result[i].Status = RenewCheckStatusNoExist
				resp.Result[i].Message = "no exist"
			}
			continue
		}

		if now-int64(acc.ExpiredAt) > 0 {
			isOk = false
			if now-int64(acc.ExpiredAt) <= int64(expirationGracePeriod) {
				resp.Result[i].Status = RenewCheckStatusExpirationGracePeriod
				resp.Result[i].Message = "expiration grace period"
			} else {
				resp.Result[i].Status = RenewCheckStatusExpired
				resp.Result[i].Message = "account expired"
			}
		}
	}
	return isOk, &resp, nil
}