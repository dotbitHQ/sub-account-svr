package handle

import (
	"context"
	"das_sub_account/tables"
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

type ReqPadgeRecordEdit struct {
	List []ReqPadgeRecordEditData `json:"list"`
}

type ReqPadgeRecordEditData struct {
	Account     string                   `json:"account"`
	Nonce       uint64                   `json:"nonce"`
	Signature   string                   `json:"signature"`
	SignAddress string                   `json:"sign_address"`
	ExpiredAt   uint64                   `json:"expired_at"`
	AlgId       common.DasAlgorithmId    `json:"alg_id"`
	SubAlgId    common.DasSubAlgorithmId `json:"sub_alg_id"`
	Payload     string                   `json:"payload"`
	Records     []witness.Record         `json:"records"`
}

type RespPadgeRecordEdit struct {
}

func (h *HttpHandle) PadgeRecordEdit(ctx *gin.Context) {
	var (
		funcName               = "PadgeRecordEdit"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqPadgeRecordEdit
		apiResp                http_api.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req), ctx.Request.Context())

	//time.Sleep(time.Minute * 3)
	if err = h.doPadgeRecordEdit(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doPadgeRecordEdit err:", err.Error(), funcName, clientIp, ctx.Request.Context())
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doPadgeRecordEdit(ctx context.Context, req *ReqPadgeRecordEdit, apiResp *http_api.ApiResp) error {
	var resp RespPadgeRecordEdit

	var list []tables.TableSmtRecordInfo
	for _, v := range req.List {
		accountId := common.Bytes2Hex(common.GetAccountIdByAccount(v.Account))
		acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
		if err != nil {
			apiResp.ApiRespErr(http_api.ApiCodeError500, err.Error())
			return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
		} else if acc.Id == 0 {
			apiResp.ApiRespErr(http_api.ApiCodeAccountNotExist, "account not exist")
			return nil
		} else if acc.ParentAccountId != "0x71cb663835a96d62020647e0bde504968558d5e6" {
			apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, "account is invalid")
			return nil
		}

		payloadBys, _ := hex.DecodeString(v.Payload)
		addressHex := common.FormatAddressPayload(payloadBys, v.AlgId)
		addrNormal, err := h.DasCore.Daf().HexToNormal(core.DasAddressHex{
			DasAlgorithmId:    v.AlgId,
			DasSubAlgorithmId: v.SubAlgId,
			AddressHex:        addressHex,
			AddressPayload:    payloadBys,
			IsMulti:           false,
			ChainType:         v.AlgId.ToChainType(),
		})

		// add record
		smtRecord := tables.TableSmtRecordInfo{
			AccountId:       accountId,
			Nonce:           v.Nonce + 1,
			RecordType:      tables.RecordTypeDefault,
			Action:          common.DasActionUpdateSubAccount,
			ParentAccountId: acc.ParentAccountId,
			Account:         acc.Account,
			EditKey:         "records",
			Signature:       v.Signature,
			LoginChainType:  v.AlgId.ToChainType(),
			LoginAddress:    addrNormal.AddressNormal,
			SignAddress:     v.SignAddress,
			EditRecords:     toolib.JsonString(&v.Records),
			Timestamp:       time.Now().UnixMilli(),
			SubAction:       common.SubActionEdit,
			ExpiredAt:       v.ExpiredAt,
		}
		list = append(list, smtRecord)

	}
	if len(list) > 0 {
		if err := h.DbDao.CreateSmtRecordList(list); err != nil {
			apiResp.ApiRespErr(http_api.ApiCodeDbError, "fail to create smt record")
			return fmt.Errorf("CreateSmtRecordList err:%s", err.Error())
		}
	}

	apiResp.ApiRespOK(resp)
	return nil
}
