package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqAccountDetail struct {
	Account string `json:"account"`
}

type RespAccountDetail struct {
	AccountInfo AccountData  `json:"account_info"`
	Records     []RecordData `json:"records"`
}

type AccountData struct {
	Account              string                    `json:"account"`
	Owner                api_code.ChainTypeAddress `json:"owner"`
	Manager              api_code.ChainTypeAddress `json:"manager"`
	RegisteredAt         int64                     `json:"registered_at"`
	ExpiredAt            int64                     `json:"expired_at"`
	Status               tables.AccountStatus      `json:"status"`
	EnableSubAccount     tables.EnableSubAccount   `json:"enable_sub_account"`
	RenewSubAccountPrice uint64                    `json:"renew_sub_account_price"`
	Nonce                uint64                    `json:"nonce"`
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
		funcName = "AccountDetail"
		clientIp = GetClientIp(ctx)
		req      ReqAccountDetail
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

	if err = h.doAccountDetail(&req, &apiResp); err != nil {
		log.Error("doAccountDetail err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAccountDetail(req *ReqAccountDetail, apiResp *api_code.ApiResp) error {
	var resp RespAccountDetail
	resp.Records = make([]RecordData, 0)

	// check params
	if req.Account == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid")
		return nil
	}

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

	// get records
	list, err := h.DbDao.GetRecordsByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query records")
		return fmt.Errorf("GetRecordsByAccountId err: %s", err.Error())
	}
	for _, v := range list {
		tmp := recordsInfoToRecordData(v)
		resp.Records = append(resp.Records, tmp)
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) accountInfoToAccountData(acc tables.TableAccountInfo) AccountData {
	var owner, manager api_code.ChainTypeAddress

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

	owner = api_code.FormatChainTypeAddress(config.Cfg.Server.Net, acc.OwnerChainType, ownerHex.AddressNormal)
	manager = api_code.FormatChainTypeAddress(config.Cfg.Server.Net, acc.ManagerChainType, managerHex.AddressNormal)

	return AccountData{
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
