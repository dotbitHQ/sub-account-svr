package handle

import (
	"bytes"
	"context"
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
)

type ReqAccountDetail struct {
	Account string `json:"account"`
}

type RespAccountDetail struct {
	AccountInfo  AccountData  `json:"account_info"`
	Records      []RecordData `json:"records"`
	CustomScript string       `json:"custom_script"`
}

type AccountData struct {
	AccountId            string                  `json:"account_id"`
	Account              string                  `json:"account"`
	Owner                core.ChainTypeAddress   `json:"owner"`
	Manager              core.ChainTypeAddress   `json:"manager"`
	RegisteredAt         int64                   `json:"registered_at"`
	ExpiredAt            int64                   `json:"expired_at"`
	Status               tables.AccountStatus    `json:"status"`
	EnableSubAccount     tables.EnableSubAccount `json:"enable_sub_account"`
	RenewSubAccountPrice uint64                  `json:"renew_sub_account_price"`
	Nonce                uint64                  `json:"nonce"`
	Avatar               string                  `json:"avatar"`
}

type RecordData struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Label string `json:"label"`
	Value string `json:"value"`
	Ttl   string `json:"ttl"`
}

func (h *HttpHandle) AccountDetail(ctx *gin.Context) {
	var (
		funcName               = "AccountDetail"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqAccountDetail
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx.Request.Context())

	if err = h.doAccountDetail(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doAccountDetail err:", err.Error(), funcName, clientIp, ctx.Request.Context())
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAccountDetail(ctx context.Context, req *ReqAccountDetail, apiResp *api_code.ApiResp) error {
	var resp RespAccountDetail
	resp.Records = make([]RecordData, 0)

	// check params
	if req.Account == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid")
		return nil
	}
	req.Account = strings.ToLower(req.Account)

	// get account detail
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
	} else if acc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return nil
	}
	resp.AccountInfo = h.accountInfoToAccountData(acc)

	// custom-script
	if acc.EnableSubAccount == tables.AccountEnableStatusOn {
		subAccLiveCell, err := h.DasCore.GetSubAccountCell(acc.AccountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return nil
		}
		detailSub := witness.ConvertSubAccountCellOutputData(subAccLiveCell.OutputData)
		defaultCS := make([]byte, 32)
		if len(detailSub.CustomScriptArgs) > 0 && bytes.Compare(defaultCS, detailSub.CustomScriptArgs) != 0 {
			resp.CustomScript = common.Bytes2Hex(detailSub.CustomScriptArgs)
		}
	}

	// get records
	list, err := h.DbDao.GetRecordsByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query records")
		return fmt.Errorf("GetRecordsByAccountId err: %s", err.Error())
	}
	for _, v := range list {
		tmp := recordsInfoToRecordData(v)
		resp.Records = append(resp.Records, tmp)
		if v.Type == "profile" && v.Key == "avatar" {
			resp.AccountInfo.Avatar = v.Value
		}
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) accountInfoToAccountData(acc tables.TableAccountInfo) AccountData {
	var owner, manager core.ChainTypeAddress

	ownerHex, _ := h.DasCore.Daf().HexToNormal(core.DasAddressHex{
		DasAlgorithmId: acc.OwnerChainType.ToDasAlgorithmId(true),
		AddressHex:     acc.Owner,
		IsMulti:        false,
		ChainType:      acc.OwnerChainType,
	})
	managerHex, _ := h.DasCore.Daf().HexToNormal(core.DasAddressHex{
		DasAlgorithmId: acc.ManagerChainType.ToDasAlgorithmId(true),
		AddressHex:     acc.Manager,
		IsMulti:        false,
		ChainType:      acc.ManagerChainType,
	})

	owner = core.FormatChainTypeAddress(config.Cfg.Server.Net, acc.OwnerChainType, ownerHex.AddressNormal)
	manager = core.FormatChainTypeAddress(config.Cfg.Server.Net, acc.ManagerChainType, managerHex.AddressNormal)

	return AccountData{
		AccountId:            acc.AccountId,
		Account:              acc.Account,
		Owner:                owner,
		Manager:              manager,
		RegisteredAt:         int64(acc.RegisteredAt) * 1e3,
		ExpiredAt:            int64(acc.ExpiredAt) * 1e3,
		Status:               acc.Status,
		EnableSubAccount:     acc.EnableSubAccount,
		RenewSubAccountPrice: acc.RenewSubAccountPrice,
		Nonce:                acc.Nonce,
	}
}

func recordsInfoToRecordData(r tables.TableRecordsInfo) RecordData {
	return RecordData{
		Key:   r.Key,
		Type:  r.Type,
		Label: r.Label,
		Value: r.Value,
		Ttl:   r.Ttl,
	}
}
