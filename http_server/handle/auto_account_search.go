package handle

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"math"
	"net/http"
	"strings"
	"time"
)

type ReqAutoAccountSearch struct {
	core.ChainTypeAddress
	ActionType tables.ActionType `json:"action_type"`
	SubAccount string            `json:"sub_account"`
}

type RespAutoAccountSearch struct {
	Price             decimal.Decimal `json:"price"`
	MaxYear           uint64          `json:"max_year"`
	Status            AccStatus       `json:"status"`
	IsSelf            bool            `json:"is_self"`
	OrderId           string          `json:"order_id"`
	ExpiredAt         uint64          `json:"expired_at"`
	PremiumPercentage decimal.Decimal `json:"premium_percentage"`
	PremiumBase       decimal.Decimal `json:"premium_base"`
	DefaultRenewRule  bool            `json:"default_renew_rule"`
}

type AccStatus int

const (
	AccStatusDefault  AccStatus = 0
	AccStatusMinting  AccStatus = 1
	AccStatusMinted   AccStatus = 2
	AccStatusRenewing AccStatus = 3
	AccStatusUnMinted AccStatus = 4
	AccStatusExpired  AccStatus = 5
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
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx)

	if err = h.doAutoAccountSearch(&req, &apiResp); err != nil {
		log.Error("doAutoAccountSearch err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoAccountSearch(req *ReqAutoAccountSearch, apiResp *api_code.ApiResp) error {
	var resp RespAutoAccountSearch
	// check sub_account name
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

	// check sub_account
	subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccount))
	hexAddr, _ := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	resp.Status, resp.IsSelf, resp.OrderId, resp.ExpiredAt, err = h.checkSubAccount(req.ActionType, apiResp, hexAddr, subAccountId)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check switch
	resp.DefaultRenewRule, err = h.checkSwitch(parentAccountId, req.ActionType, apiResp)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// get max years
	resp.MaxYear = h.getMaxYears(parentAccount)

	// check min price 0.99$
	builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to get config info")
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	// get rule price
	defaultRenewRule := false
	resp.Price, defaultRenewRule, err = h.getRulePrice(parentAccount.Account, parentAccountId, req.SubAccount, apiResp, req.ActionType, builder)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	if !resp.DefaultRenewRule {
		resp.DefaultRenewRule = defaultRenewRule
	}

	newSubAccountPrice, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.NewSubAccountPrice().RawData())
	minPrice := decimal.NewFromInt(int64(newSubAccountPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
	if req.ActionType == tables.ActionTypeRenew {
		renewSubAccountPrice, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.RenewSubAccountPrice().RawData())
		minPrice = decimal.NewFromInt(int64(renewSubAccountPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
	}
	if minPrice.GreaterThan(resp.Price) {
		apiResp.ApiRespErr(api_code.ApiCodePriceRulePriceNotBeLessThanMin, "Pricing below minimum")
		return fmt.Errorf("price not be less than min: %s$", minPrice.String())
	}

	resp.PremiumPercentage = config.Cfg.Stripe.PremiumPercentage
	resp.PremiumBase = config.Cfg.Stripe.PremiumBase

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

	//
	configCellBuilder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "failed to get config cell account")
		return
	}
	maxLength, _ := configCellBuilder.MaxLength()
	accountCharStr, err := h.DasCore.GetAccountCharSetList(subAccountName)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "failed to get account charset list")
		return
	}
	accLen := len(accountCharStr)
	if uint32(accLen) > maxLength {
		apiResp.ApiRespErr(api_code.ApiCodeExceededMaxLength, fmt.Sprintf("Exceeded the max length of the sub-account: %d", maxLength))
		return
	}
	if !h.checkAccountCharSet(accountCharStr, subAccountName[:strings.Index(subAccountName, ".")]) {
		log.Info("checkAccountCharSet:", subAccountName, accountCharStr)
		apiResp.ApiRespErr(api_code.ApiCodeInvalidCharset, fmt.Sprintf("sub-account[%s] invalid", subAccountName))
		return
	}

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
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountExpired, "parent account is expired")
		return nil, nil
	}
	expiredAt := uint64(time.Now().Add(time.Hour * 24 * 7).Unix())
	if expiredAt > parentAccount.ExpiredAt {
		apiResp.ApiRespErr(api_code.ApiCodeAccountExpiringSoon, "account expiring soon")
		return nil, nil
	}
	return &parentAccount, nil
}

func (h *HttpHandle) checkSubAccount(actionType tables.ActionType, apiResp *api_code.ApiResp, hexAddr *core.DasAddressHex, subAccountId string) (accStatus AccStatus, isSelf bool, orderId string, expiredAt uint64, e error) {
	accStatus = AccStatusDefault
	subAccount, err := h.DbDao.GetAccountInfoByAccountId(subAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query sub-account")
		e = fmt.Errorf("GetAccountInfoByAccountId err: %s %s", err.Error(), subAccountId)
		return
	}
	if actionType == tables.ActionTypeMint && subAccount.Id > 0 {
		accStatus = AccStatusMinted
		return
	}

	if actionType == tables.ActionTypeRenew {
		if subAccount.Id == 0 {
			accStatus = AccStatusUnMinted
			return
		}

		configCellBuilder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsAccount)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "failed to get config cell account")
			e = fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
			return
		}
		expirationGracePeriod, err := configCellBuilder.ExpirationGracePeriod()
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			e = err
			return
		}
		if time.Now().Unix()-int64(subAccount.ExpiredAt) > int64(expirationGracePeriod) {
			accStatus = AccStatusExpired
			return
		}
		expiredAt = subAccount.ExpiredAt * 1e3
	}

	subAction := common.SubActionCreate
	if actionType == tables.ActionTypeRenew {
		subAction = common.SubActionRenew
	}
	smtRecord, err := h.DbDao.GetLatestSmtRecordAccountId(subAccountId, subAction)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query mint record")
		e = fmt.Errorf("GetLatestSmtRecordAccountId err: %s %s", err.Error(), subAccountId)
		return
	}

	// check smt record pending
	if smtRecord.Id > 0 && smtRecord.RecordType == tables.RecordTypeDefault {
		switch subAction {
		case common.SubActionCreate:
			accStatus = AccStatusMinting
		case common.SubActionRenew:
			accStatus = AccStatusRenewing
		}
		return
	}

	if hexAddr != nil {
		// check order of self
		orderInfo, err := h.DbDao.GetMintOrderInProgressByAccountIdWithAddr(subAccountId, hexAddr.AddressHex, actionType)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order")
			e = fmt.Errorf("GetMintOrderInProgressByAccountIdWithAddr err: %s %s", err.Error(), subAccountId)
			return
		}
		if orderInfo.Id > 0 {
			if smtRecord.OrderID == orderInfo.OrderId && orderInfo.OrderStatus != tables.OrderStatusDefault {
				accStatus = AccStatusDefault
				return
			}
			isSelf, orderId = true, orderInfo.OrderId

			switch actionType {
			case tables.ActionTypeMint:
				accStatus = AccStatusMinting
			case tables.ActionTypeRenew:
				accStatus = AccStatusRenewing
			}
			return
		}
	}

	// check order of others
	orderInfo, err := h.DbDao.GetMintOrderInProgressByAccountId(subAccountId, actionType)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query order")
		e = fmt.Errorf("GetMintOrderInProgressByAccountIdWithAddr err: %s %s", err.Error(), subAccountId)
		return
	}
	if orderInfo.Id > 0 && orderInfo.OrderStatus == tables.OrderStatusDefault {
		switch actionType {
		case tables.ActionTypeMint:
			accStatus = AccStatusMinting
		case tables.ActionTypeRenew:
			accStatus = AccStatusRenewing
		}
	}
	return
}

func (h *HttpHandle) checkSwitch(parentAccountId string, actionType tables.ActionType, apiResp *api_code.ApiResp) (bool, error) {
	subAccCell, err := h.DasCore.GetSubAccountCell(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return false, fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}
	subAccTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return false, fmt.Errorf("GetTransaction err: %s", err.Error())
	}
	subAccData := witness.ConvertSubAccountCellOutputData(subAccTx.Transaction.OutputsData[subAccCell.OutPoint.Index])
	if subAccData.AutoDistribution == witness.AutoDistributionDefault {
		if actionType == tables.ActionTypeRenew {
			return true, nil
		}
		apiResp.ApiRespErr(api_code.ApiCodeAutoDistributionClosed, "Automatic allocation is not turned on")
		return false, nil
	}
	return false, nil
}

func (h *HttpHandle) getMaxYears(parentAccount *tables.TableAccountInfo) uint64 {
	nowT := uint64(time.Now().Unix())
	if nowT > parentAccount.ExpiredAt {
		return 0
	}
	maxYear := (parentAccount.ExpiredAt - nowT) / uint64(common.OneYearSec)
	if maxYear == 0 {
		return 1
	}
	log.Info("getMaxYears:", parentAccount.ExpiredAt, maxYear, config.Cfg.Das.MaxRegisterYears)
	if maxYear > config.Cfg.Das.MaxRegisterYears {
		maxYear = config.Cfg.Das.MaxRegisterYears
	}
	return maxYear
}

func (h *HttpHandle) getRulePrice(parentAcc, parentAccountId, subAccount string, apiResp *api_code.ApiResp, actionType tables.ActionType, priceBuilder *witness.ConfigCellDataBuilder) (price decimal.Decimal, defaultRenewRule bool, e error) {
	ruleConfig, err := h.DbDao.GetRuleConfigByAccountId(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to search rule config")
		e = fmt.Errorf("GetRuleConfigByAccountId err: %s", err.Error())
		return
	} else if ruleConfig.TxHash == "" {
		if actionType == tables.ActionTypeRenew {
			defaultRenewRule = true
			renewSubAccountPrice, _ := molecule.Bytes2GoU64(priceBuilder.ConfigCellSubAccount.RenewSubAccountPrice().RawData())
			price = decimal.NewFromInt(int64(renewSubAccountPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
			return
		}
	}
	ruleTx, err := h.DasCore.Client().GetTransaction(h.Ctx, types.HexToHash(ruleConfig.TxHash))
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to search rule tx")
		e = fmt.Errorf("GetTransaction err: %s", err.Error())
		return
	}
	ruleReverse := witness.NewSubAccountRuleEntity(parentAcc)
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

	rulePrice := witness.NewSubAccountRuleEntity(parentAcc)
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
		if actionType == tables.ActionTypeRenew {
			defaultRenewRule = true
			renewSubAccountPrice, _ := molecule.Bytes2GoU64(priceBuilder.ConfigCellSubAccount.RenewSubAccountPrice().RawData())
			price = decimal.NewFromInt(int64(renewSubAccountPrice)).DivRound(decimal.NewFromInt(common.UsdRateBase), 2)
			return
		}
		apiResp.ApiRespErr(api_code.ApiCodeNoTSetRules, "not set price rules")
		return
	}
	price = decimal.NewFromInt(int64(rulePrice.Rules[index].Price)).Div(decimal.NewFromFloat(math.Pow10(6)))
	return
}
