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
	"github.com/scorpiotzh/toolib"
	"golang.org/x/sync/errgroup"
	"net/http"
	"time"
)

type ReqSubAccountRenew struct {
	core.ChainTypeAddress
	chainType      common.ChainType
	address        string
	Account        string            `json:"account"`
	SubAccountList []RenewSubAccount `json:"sub_account_list"`
}

type RenewSubAccount struct {
	Account        string                  `json:"account"`
	AccountCharStr []common.AccountCharSet `json:"account_char_str"`
	RenewYears     uint64                  `json:"renew_years"`
}

type RespSubAccountRenew struct {
	SignInfoList
}

func (h *HttpHandle) SubAccountRenew(ctx *gin.Context) {
	var (
		funcName               = "SubAccountRenew"
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

	if err = h.doSubAccountRenew(&req, &apiResp); err != nil {
		log.Error("doSubAccountRenew err:", err.Error(), funcName, clientIp)
		doApiError(err, &apiResp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountRenew(req *ReqSubAccountRenew, apiResp *api_code.ApiResp) error {

	return nil
}

func (h *HttpHandle) doSubAccountRenewCheckParams(req *ReqSubAccountRenew, apiResp *api_code.ApiResp) error {
	if lenList := len(req.SubAccountList); lenList == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: len(sub account list) is 0")
		return nil
	} else if lenList > config.Cfg.Das.MaxRenewCount {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("more than max renew num %d", config.Cfg.Das.MaxCreateCount))
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

func (h *HttpHandle) doSubAccountRenewCheckAccount(req *ReqSubAccountRenew, apiResp *api_code.ApiResp) (*tables.TableAccountInfo, error) {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account")
		return nil, fmt.Errorf("GetAccountInfoByAccountId: %s", err.Error())
	}
	if acc.Id == 0 {
		err = errors.New("account not exist")
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, err.Error())
		return nil, err
	}
	if acc.EnableSubAccount != tables.AccountEnableStatusOn {
		err = errors.New("sub account uninitialized")
		apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOff, err.Error())
		return nil, err
	}

	// config cell
	builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount, common.ConfigCellTypeArgsAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return nil, fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	expirationGracePeriod, err := builder.ExpirationGracePeriod()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return nil, err
	}
	if time.Now().Unix()-int64(acc.ExpiredAt) > int64(expirationGracePeriod) {
		err = errors.New("account expired")
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, err.Error())
		return nil, err
	}

	// check sub_account
	wg := &errgroup.Group{}
	wg.SetLimit(10)
	for _, v := range req.SubAccountList {
		account := v.Account
		wg.Go(func() error {
			subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
			subAcc, err := h.DbDao.GetAccountInfoByAccountId(subAccountId)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query sub account")
				return fmt.Errorf("GetAccountInfoByAccountId: %s", err.Error())
			}
			if subAcc.Id == 0 {
				err = fmt.Errorf("sub account: %s no exist", subAcc.Account)
				apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, err.Error())
				return err
			}
			if time.Now().Unix()-int64(subAcc.ExpiredAt) > int64(expirationGracePeriod) {
				err = errors.New("account expired")
				apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, err.Error())
				return err
			}
			return nil
		})
	}
	if err := wg.Wait(); err != nil {
		return nil, err
	}
	return &acc, nil
}
