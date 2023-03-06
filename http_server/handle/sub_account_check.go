package handle

import (
	"bytes"
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/witness"
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
		funcName               = "SubAccountCheck"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSubAccountCreate
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
	_, err := h.doSubAccountCheckAccount(req.Account, apiResp, common.DasActionUpdateSubAccount)
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
	addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, true)
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
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "account status is not normal")
		return nil, nil
	} else if acc.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, "account is expired")
		return nil, nil
	}
	switch action {
	case common.DasActionUpdateSubAccount:
		if acc.EnableSubAccount != tables.AccountEnableStatusOn {
			apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOff, "sub account uninitialized")
			return nil, nil
		}
	case common.DasActionEnableSubAccount:
		if acc.EnableSubAccount == tables.AccountEnableStatusOn {
			apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOn, "sub account already initialized")
			return nil, nil
		}
	default:
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("unknow action[%s]", action))
		return nil, nil
	}
	return &acc, nil
}

func (h *HttpHandle) doSubAccountCheckList(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) (bool, *RespSubAccountCheck, error) {
	isOk := true
	var resp RespSubAccountCheck
	resp.Result = make([]CheckSubAccount, 0)

	// check mint for account
	if err := h.doMintForAccountCheck(req, apiResp); err != nil {
		return false, nil, fmt.Errorf("doMintForAccountCheck err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return false, nil, nil
	}

	//
	var subAccountMap = make(map[string]int)
	configCellBuilder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "failed to get config cell account")
		return false, nil, fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	maxLength, _ := configCellBuilder.MaxLength()
	// check list
	var accountIds []string
	for i, _ := range req.SubAccountList {
		tmp := CheckSubAccount{
			CreateSubAccount: req.SubAccountList[i],
			Status:           0,
			Message:          "",
		}
		index := strings.Index(req.SubAccountList[i].Account, ".")
		if index == -1 {
			tmp.Status = CheckStatusFail
			tmp.Message = fmt.Sprintf("sub account invalid: %s", req.SubAccountList[i].Account)
			isOk = false
			resp.Result = append(resp.Result, tmp)
			continue
		}
		//
		suffix := strings.TrimLeft(req.SubAccountList[i].Account[index:], ".")
		if suffix != req.Account {
			tmp.Status = CheckStatusFail
			tmp.Message = fmt.Sprintf("account suffix diff: %s", suffix)
			isOk = false
			resp.Result = append(resp.Result, tmp)
			continue
		}
		//
		accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccountList[i].Account))

		if len(req.SubAccountList[i].AccountCharStr) == 0 {
			accountCharStr, err := h.DasCore.GetAccountCharSetList(req.SubAccountList[i].Account)
			//accountCharStr, err := common.AccountToAccountChars(v.Account)
			if err != nil {
				tmp.Status = CheckStatusFail
				tmp.Message = fmt.Sprintf("AccountToAccountChars err: %s", suffix)
				isOk = false
				resp.Result = append(resp.Result, tmp)
				continue
			}
			req.SubAccountList[i].AccountCharStr = accountCharStr
		}

		accLen := len(req.SubAccountList[i].AccountCharStr)
		if uint32(accLen) > maxLength {
			tmp.Status = CheckStatusFail
			tmp.Message = fmt.Sprintf("account len more than: %d", maxLength)
			isOk = false
		} else if indexAcc, ok := subAccountMap[accountId]; ok {
			resp.Result[indexAcc].Status = CheckStatusFail
			resp.Result[indexAcc].Message = fmt.Sprintf("same account")
			tmp.Status = CheckStatusFail
			tmp.Message = fmt.Sprintf("same account")
			isOk = false
		} else if req.SubAccountList[i].RegisterYears <= 0 {
			tmp.Status = CheckStatusFail
			tmp.Message = "register years less than 1"
			isOk = false
		} else if req.SubAccountList[i].RegisterYears > config.Cfg.Das.MaxRegisterYears {
			tmp.Status = CheckStatusFail
			tmp.Message = fmt.Sprintf("register years more than %d", config.Cfg.Das.MaxRegisterYears)
			isOk = false
		} else if !h.checkAccountCharSet(req.SubAccountList[i].AccountCharStr, req.SubAccountList[i].Account[:strings.Index(req.SubAccountList[i].Account, ".")]) {
			log.Info("checkAccountCharSet:", req.SubAccountList[i].Account, req.SubAccountList[i].AccountCharStr)
			tmp.Status = CheckStatusFail
			tmp.Message = fmt.Sprintf("checkAccountCharSet invalid charset")
			isOk = false
		}
		if tmp.Status != CheckStatusOk {
			resp.Result = append(resp.Result, tmp)
			continue
		}
		//
		addrHex, e := req.SubAccountList[i].FormatChainTypeAddress(config.Cfg.Server.Net, true)
		if e != nil {
			tmp.Status = CheckStatusFail
			tmp.Message = fmt.Sprintf("params is invalid: %s", e.Error())
			isOk = false
		} else {
			accId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccountList[i].Account))
			accountIds = append(accountIds, accId)
			req.SubAccountList[i].chainType, req.SubAccountList[i].address = addrHex.ChainType, addrHex.AddressHex
			subAccountMap[accountId] = i
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

func (h *HttpHandle) doMintForAccountCheck(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	var mintForAccountIds []string
	for i, _ := range req.SubAccountList {
		if req.SubAccountList[i].KeyInfo.Key != "" {
			continue
		}
		accId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccountList[i].MintForAccount))
		mintForAccountIds = append(mintForAccountIds, accId)
	}
	mintForAccountList, err := h.DbDao.GetAccountListByAccountIds(mintForAccountIds)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query mint for account list")
		return fmt.Errorf("GetAccountListByAccountIds: %s", err.Error())
	}
	var mapMinForAccount = make(map[string]tables.TableAccountInfo)
	for i, v := range mintForAccountList {
		mapMinForAccount[v.AccountId] = mintForAccountList[i]
	}
	for i, _ := range req.SubAccountList {
		if req.SubAccountList[i].KeyInfo.Key != "" {
			continue
		}
		accId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccountList[i].MintForAccount))
		if acc, ok := mapMinForAccount[accId]; ok {
			coinType := common.CoinTypeEth
			keyOwner := acc.Owner
			if acc.OwnerAlgorithmId == common.DasAlgorithmIdTron {
				coinType = common.CoinTypeTrx
				keyOwner, _ = common.TronHexToBase58(acc.Owner)
			}
			req.SubAccountList[i].ChainTypeAddress.Type = "blockchain"
			req.SubAccountList[i].ChainTypeAddress.KeyInfo.CoinType = coinType
			req.SubAccountList[i].ChainTypeAddress.KeyInfo.Key = keyOwner
		} else {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("mint for account [%s] invalid", req.SubAccountList[i].MintForAccount))
			return fmt.Errorf("mint for account [%s] invalid", req.SubAccountList[i].MintForAccount)
		}
	}
	return nil
}

func (h *HttpHandle) doSubAccountCheckCustomScript(parentAccountId string, req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	defaultCustomScriptArgs := make([]byte, 33)
	subAccountLiveCell, err := h.DasCore.GetSubAccountCell(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}
	subDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
	if len(subDetail.CustomScriptArgs) == 0 || bytes.Compare(subDetail.CustomScriptArgs, defaultCustomScriptArgs) == 0 {
		return nil
	}

	builderConfigCellSub, err := h.DasCore.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgs err: %s", err.Error())
	}
	newSubAccountPrice, _ := molecule.Bytes2GoU64(builderConfigCellSub.ConfigCellSubAccount.NewSubAccountPrice().RawData())
	quoteCell, err := h.DasCore.GetQuoteCell()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetQuoteCell err: %s", err.Error())
	}
	quote := quoteCell.Quote()

	priceApi := txtool.PriceApiConfig{
		DasCore: h.DasCore,
		DbDao:   h.DbDao,
	}
	totalCKB := uint64(0)
	minDasCKb := uint64(0)
	for _, v := range req.SubAccountList {
		resPrice, err := priceApi.GetPrice(&txtool.ParamGetPrice{
			Action:         common.DasActionUpdateSubAccount,
			SubAction:      common.SubActionCreate,
			SubAccount:     v.Account,
			RegisterYears:  v.RegisterYears,
			AccountCharStr: v.AccountCharStr,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("GetPrice err: %s", err.Error())
		}
		priceCkb := (resPrice.ActionTotalPrice / quote) * common.OneCkb
		totalCKB += priceCkb
		minDasCKb += v.RegisterYears * newSubAccountPrice
	}
	if totalCKB < minDasCKb {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "price invalid")
		return nil
	}
	return nil
}

func (h *HttpHandle) checkAccountCharSet(accountCharSet []common.AccountCharSet, account string) bool {
	var accountCharStr string
	for _, v := range accountCharSet {
		if v.Char == "" {
			return false
		}
		switch v.CharSetName {
		case common.AccountCharTypeEmoji:
			if _, ok := common.CharSetTypeEmojiMap[v.Char]; !ok {
				return false
			}
		case common.AccountCharTypeDigit:
			if _, ok := common.CharSetTypeDigitMap[v.Char]; !ok {
				return false
			}
		case common.AccountCharTypeEn:
			if _, ok := common.CharSetTypeEnMap[v.Char]; v.Char != "." && !ok {
				return false
			}
		case common.AccountCharTypeJa:
			if _, ok := common.CharSetTypeJaMap[v.Char]; !ok {
				return false
			}
		case common.AccountCharTypeRu:
			if _, ok := common.CharSetTypeRuMap[v.Char]; !ok {
				return false
			}
		case common.AccountCharTypeTr:
			if _, ok := common.CharSetTypeTrMap[v.Char]; !ok {
				return false
			}
		case common.AccountCharTypeVi:
			if _, ok := common.CharSetTypeViMap[v.Char]; !ok {
				return false
			}
		case common.AccountCharTypeTh:
			if _, ok := common.CharSetTypeThMap[v.Char]; !ok {
				return false
			}
		case common.AccountCharTypeKo:
			if _, ok := common.CharSetTypeKoMap[v.Char]; !ok {
				return false
			}
		default:
			return false
		}
		accountCharStr += v.Char
	}
	if !strings.EqualFold(accountCharStr, account) {
		return false
	}
	if isDiff := common.CheckAccountCharTypeDiff(accountCharSet); isDiff {
		return false
	}
	return true
}
