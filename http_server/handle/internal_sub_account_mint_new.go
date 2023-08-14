package handle

import (
	"bytes"
	"das_sub_account/config"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

func (h *HttpHandle) InternalSubAccountMintNew(ctx *gin.Context) {
	var (
		funcName               = "InternalSubAccountMintNew"
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

	if err = h.doInternalSubAccountMintNew(&req, &apiResp); err != nil {
		log.Error("doInternalSubAccountMintNew err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

type RespInternalSubAccountMintNew struct{}

func (h *HttpHandle) doInternalSubAccountMintNew(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	var resp RespInternalSubAccountMintNew
	// check params
	if err := h.doSubAccountCheckParams(req, apiResp); err != nil {
		return fmt.Errorf("doSubAccountCheckParams err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check account
	acc, err := h.doSubAccountCheckAccount(req.Account, apiResp, common.DasActionUpdateSubAccount)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckAccount err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check list
	isOk, respCheck, err := h.doSubAccountCheckList(req, apiResp)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckList err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if !isOk {
		log.Error("doSubAccountCheckList:", toolib.JsonString(respCheck))
		apiResp.ApiRespErr(api_code.ApiCodeCreateListCheckFail, "create list check failed")
		return nil
	}

	// check account price
	if err := h.doSubAccountCheckCustomScriptNew(acc, req, apiResp); err != nil {
		return fmt.Errorf("doSubAccountCheckCustomScriptNew err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// do distribution
	parentAccountId := acc.AccountId
	if _, ok := config.Cfg.SuspendMap[parentAccountId]; ok {
		apiResp.ApiRespErr(api_code.ApiCodeSuspendOperation, "suspend operation")
		return nil
	}

	recordList, err := getRecordListNew(h.DasCore.Daf(), req, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("getRecordListNew err: %s", err.Error())
	}

	if err := h.DbDao.CreateSmtRecordList(recordList); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return fmt.Errorf("CreateSmtRecordList err: %s", err.Error())
	}

	apiResp.ApiRespOK(resp)

	return nil
}

func (h *HttpHandle) doSubAccountCheckCustomScriptNew(acc *tables.TableAccountInfo, req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	defaultCustomScriptArgs := make([]byte, 32)
	subAccountLiveCell, err := h.DasCore.GetSubAccountCell(acc.AccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}

	subDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
	if len(subDetail.CustomScriptArgs) == 0 || bytes.Compare(subDetail.CustomScriptArgs, defaultCustomScriptArgs) == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "CustomScriptArgs is nil")
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

func getRecordListNew(daf *core.DasAddressFormat, req *ReqSubAccountCreate, parentAccountId string) ([]tables.TableSmtRecordInfo, error) {
	var recordList []tables.TableSmtRecordInfo

	for _, v := range req.SubAccountList {
		subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(v.Account))
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: v.chainType.ToDasAlgorithmId(true),
			AddressHex:     v.address,
			IsMulti:        false,
			ChainType:      v.chainType,
		}
		registerArgs, err := daf.HexToArgs(ownerHex, ownerHex)
		if err != nil {
			return nil, fmt.Errorf("HexToArgs err: %s", err.Error())
		}
		var content []byte
		if len(v.AccountCharStr) > 0 {
			content, err = json.Marshal(v.AccountCharStr)
			if err != nil {
				return nil, fmt.Errorf("json Marshal err: %s", err.Error())
			}
		}
		smtRecord := tables.TableSmtRecordInfo{
			Id:              0,
			SvrName:         config.Cfg.Slb.SvrName,
			AccountId:       subAccountId,
			Nonce:           0,
			RecordType:      tables.RecordTypeDefault,
			MintType:        tables.MintTypeCustomScript,
			TaskId:          "",
			Action:          common.DasActionUpdateSubAccount,
			ParentAccountId: parentAccountId,
			Account:         v.Account,
			Content:         string(content),
			RegisterYears:   v.RegisterYears,
			RegisterArgs:    common.Bytes2Hex(registerArgs),
			EditKey:         "",
			Signature:       "",
			EditArgs:        "",
			RenewYears:      0,
			EditRecords:     "",
			Timestamp:       time.Now().UnixNano() / 1e6,
			SubAction:       common.SubActionCreate,
			MintSignId:      "",
			ExpiredAt:       0,
		}
		recordList = append(recordList, smtRecord)
	}
	return recordList, nil
}
