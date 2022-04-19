package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/sign"
	"github.com/DeAccountSystems/das-lib/txbuilder"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

type ReqTransactionSend struct {
	SignInfoList
}

type RespTransactionSend struct {
	HashList []string `json:"hash_list"`
}

func (h *HttpHandle) TransactionSend(ctx *gin.Context) {
	var (
		funcName = "TransactionSend"
		clientIp = GetClientIp(ctx)
		req      ReqTransactionSend
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

	if err = h.doTransactionSend(&req, &apiResp); err != nil {
		log.Error("doTransactionSend err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doTransactionSend(req *ReqTransactionSend, apiResp *api_code.ApiResp) error {
	var resp RespTransactionSend
	resp.HashList = make([]string, 0)

	// check params
	if req.Action == "" || len(req.List) == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid")
		return nil
	}

	// check update
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}

	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	switch req.Action {
	case common.DasActionEnableSubAccount:
		var sic SignInfoCache
		if txStr, err := h.RC.GetSignTxCache(req.SignKey); err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			return fmt.Errorf("GetSignTxCache err: %s", err.Error())
		} else if err = json.Unmarshal([]byte(txStr), &sic); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			return fmt.Errorf("json.Unmarshal err: %s", err.Error())
		}
		txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, sic.BuilderTx)
		if err := txBuilder.AddSignatureForTx(req.List[0].SignList); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "add signature fail")
			return fmt.Errorf("AddSignatureForTx err: %s", err.Error())
		}
		if hash, err := txBuilder.SendTransaction(); err != nil {
			return doSendTransactionError(err, apiResp)
		} else {
			h.DasCache.AddCellInputByAction("", sic.BuilderTx.Transaction.Inputs)
			resp.HashList = append(resp.HashList, hash.Hex())
			accountId := common.Bytes2Hex(common.GetAccountIdByAccount(sic.Account))
			taskInfo := tables.TableTaskInfo{
				Id:              0,
				TaskId:          "",
				TaskType:        tables.TaskTypeNormal,
				ParentAccountId: accountId,
				Action:          common.DasActionEnableSubAccount,
				RefOutpoint:     "",
				BlockNumber:     0,
				Outpoint:        common.OutPoint2String(hash.Hex(), 1),
				Timestamp:       time.Now().UnixNano() / 1e6,
				SmtStatus:       tables.SmtStatusWriteComplete,
				TxStatus:        tables.TxStatusPending,
			}
			taskInfo.InitTaskId()
			if err := h.DbDao.CreateTask(&taskInfo); err != nil {
				log.Error("CreateTask err: ", err.Error())
			}
		}
	case common.DasActionEditSubAccount:
		var editCache EditSubAccountCache
		if txStr, err := h.RC.GetSignTxCache(req.SignKey); err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			return fmt.Errorf("GetSignTxCache err: %s", err.Error())
		} else if err = json.Unmarshal([]byte(txStr), &editCache); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			return fmt.Errorf("json.Unmarshal err: %s", err.Error())
		}
		log.Warn("EditSubAccountCache:", toolib.JsonString(&editCache))

		// check edit value
		signAddress, subAcc, err := editCache.CheckEditValue(h.DbDao, apiResp)
		if err != nil {
			return fmt.Errorf("CheckEditValue err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}

		// check now sign msg
		signData := editCache.GetSignData(subAcc, apiResp)
		if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		} else if signData.SignMsg != editCache.OldSignMsg {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "sign error")
			return nil
		}
		log.Warn("NewSignMsg:", signData.SignMsg, signAddress)

		// check sign
		data := []byte(signData.SignMsg)
		signMsg := common.Hex2Bytes(req.List[0].SignList[0].SignMsg)
		signOk := false
		switch signData.SignType {
		case common.DasAlgorithmIdEth:
			signOk, err = sign.VerifyPersonalSignature(signMsg, data, signAddress)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeSignError, "eth sign error")
				return fmt.Errorf("VerifyEthSignature err: %s", err.Error())
			}
		case common.DasAlgorithmIdTron:
			if signAddress, err = common.TronHexToBase58(signAddress); err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeSignError, "TronHexToBase58 error")
				return fmt.Errorf("TronHexToBase58 err: %s [%s]", err.Error(), signAddress)
			}
			signOk = sign.TronVerifySignature(true, signMsg, data, signAddress)
		case common.DasAlgorithmIdEd25519:
			signOk = sign.VerifyEd25519Signature(common.Hex2Bytes(signAddress), data, signMsg)
		default:
			apiResp.ApiRespErr(api_code.ApiCodeNotExistSignType, fmt.Sprintf("not exist sign type[%d]", signData.SignType))
			return nil
		}
		if !signOk {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "res sign error")
			return nil
		}

		// add record
		record := tables.TableSmtRecordInfo{
			Id:              0,
			AccountId:       subAcc.AccountId,
			Nonce:           subAcc.Nonce + 1,
			RecordType:      tables.RecordTypeDefault,
			TaskId:          "",
			Action:          common.DasActionEditSubAccount,
			ParentAccountId: subAcc.ParentAccountId,
			Account:         subAcc.Account,
			RegisterYears:   0,
			RegisterArgs:    "",
			EditKey:         editCache.EditKey,
			Signature:       req.List[0].SignList[0].SignMsg,
			EditArgs:        "",
			RenewYears:      0,
			EditRecords:     "",
			Timestamp:       time.Now().UnixNano() / 1e6,
		}

		if err := editCache.InitRecord(subAcc, &record); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return err
		}

		if err := h.DbDao.CreateSmtRecordInfo(record); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "fail to create smt record")
			return fmt.Errorf("CreateSmtRecordInfo err:%s", err.Error())
		}
	case common.DasActionCreateSubAccount:
		var signInfoCacheList SignInfoCacheList
		if txStr, err := h.RC.GetSignTxCache(req.SignKey); err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			return fmt.Errorf("GetSignTxCache err: %s", err.Error())
		} else if err = json.Unmarshal([]byte(txStr), &signInfoCacheList); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			return fmt.Errorf("json.Unmarshal err: %s", err.Error())
		}
		if len(signInfoCacheList.BuilderTxList) != len(req.List) {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "len sign list diff")
			return nil
		}
		for i, _ := range signInfoCacheList.BuilderTxList {
			txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, signInfoCacheList.BuilderTxList[i])
			if err := txBuilder.AddSignatureForTx(req.List[i].SignList); err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeError500, "add signature fail")
				return fmt.Errorf("AddSignatureForTx err: %s", err.Error())
			}
			if hash, err := txBuilder.SendTransaction(); err != nil {
				return doSendTransactionError(err, apiResp)
			} else {
				resp.HashList = append(resp.HashList, hash.Hex())
			}
			h.DasCache.AddCellInputByAction("", signInfoCacheList.BuilderTxList[i].Transaction.Inputs)
			if err := h.DbDao.UpdateTaskTxStatusToPending(signInfoCacheList.TaskIdList[i]); err != nil {
				log.Error("UpdateTaskTxStatusToPending err: ", err.Error())
			}
			time.Sleep(time.Second)
		}

	default:
		apiResp.ApiRespErr(api_code.ApiCodeNotExistConfirmAction, fmt.Sprintf("not exist confirm action[%s]", req.Action))
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}
