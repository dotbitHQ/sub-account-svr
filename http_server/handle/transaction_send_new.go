package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/txbuilder"
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

func (h *HttpHandle) TransactionSendNew(ctx *gin.Context) {
	var (
		funcName               = "TransactionSendNew"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqTransactionSend
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

	if err = h.doTransactionSendNew(&req, &apiResp); err != nil {
		log.Error("doTransactionSendNew err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doTransactionSendNew(req *ReqTransactionSend, apiResp *api_code.ApiResp) error {
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
	case common.DasActionEnableSubAccount, common.DasActionConfigSubAccountCustomScript, common.DasActionConfigSubAccount:
		if err := h.doActionNormal(req, apiResp, &resp); err != nil {
			return fmt.Errorf("doActionNormal err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}
	case common.DasActionUpdateSubAccount:
		if err := h.doActionUpdateSubAccount(req, apiResp, &resp); err != nil {
			return fmt.Errorf("doActionUpdateSubAccount err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}
	case ActionCurrencyUpdate, ActionMintConfigUpdate:
		if err := h.doActionAutoMint(req, apiResp); err != nil {
			return fmt.Errorf("doActionNormal err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}
	default:
		apiResp.ApiRespErr(api_code.ApiCodeNotExistConfirmAction, fmt.Sprintf("not exist action[%s]", req.Action))
		return nil
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doActionAutoMint(req *ReqTransactionSend, apiResp *api_code.ApiResp) error {
	switch req.Action {
	case ActionCurrencyUpdate:
		var data ReqCurrencyUpdate
		if txStr, err := h.RC.GetSignTxCache(req.SignKey); err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			return fmt.Errorf("GetSignTxCache err: %s", err.Error())
		} else if err = json.Unmarshal([]byte(txStr), &data); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			return fmt.Errorf("json.Unmarshal err: %s", err.Error())
		}
		res, err := data.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), false)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
			return fmt.Errorf("FormatChainTypeAddress err: %s", err.Error())
		}
		_, signMsg, _ := data.GetSignInfo()

		if _, err := doSignCheck(txbuilder.SignData{
			SignType: req.List[0].SignList[0].SignType,
			SignMsg:  signMsg,
		}, req.List[0].SignList[0].SignMsg, res.AddressHex, apiResp); err != nil {
			return fmt.Errorf("doSignCheck err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}

		accountId := common.Bytes2Hex(common.GetAccountIdByAccount(data.Account))
		paymentConfig, err := h.DbDao.GetUserPaymentConfig(accountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
			return err
		}
		paymentConfig.CfgMap[data.TokenId] = tables.PaymentConfigElement{
			Enable: data.Enable,
		}
		if err := h.DbDao.CreateUserConfigWithPaymentConfig(tables.UserConfig{
			Account:   data.Account,
			AccountId: accountId,
		}, paymentConfig); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to update payment config")
			return fmt.Errorf("CreateUserConfigWithMintConfig err: %s", err.Error())
		}
	case ActionMintConfigUpdate:
		var data ReqMintConfigUpdate
		if txStr, err := h.RC.GetSignTxCache(req.SignKey); err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			return fmt.Errorf("GetSignTxCache err: %s", err.Error())
		} else if err = json.Unmarshal([]byte(txStr), &data); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			return fmt.Errorf("json.Unmarshal err: %s", err.Error())
		}
		res, err := data.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), false)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
			return fmt.Errorf("FormatChainTypeAddress err: %s", err.Error())
		}
		_, signMsg, _ := data.GetSignInfo()

		if _, err := doSignCheck(txbuilder.SignData{
			SignType: req.List[0].SignList[0].SignType,
			SignMsg:  signMsg,
		}, req.List[0].SignList[0].SignMsg, res.AddressHex, apiResp); err != nil {
			return fmt.Errorf("doSignCheck err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}

		accountId := common.Bytes2Hex(common.GetAccountIdByAccount(data.Account))
		if err := h.DbDao.CreateUserConfigWithMintConfig(tables.UserConfig{
			Account:   data.Account,
			AccountId: accountId,
		}, tables.MintConfig{
			Title:           data.Title,
			Desc:            data.Desc,
			Benefits:        data.Benefits,
			Links:           data.Links,
			BackgroundColor: data.BackgroundColor,
		}); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to update mint config")
			return fmt.Errorf("CreateUserConfigWithMintConfig err: %s", err.Error())
		}
	default:
	}
	return nil
}

func (h *HttpHandle) doActionNormal(req *ReqTransactionSend, apiResp *api_code.ApiResp, resp *RespTransactionSend) error {
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
			SvrName:         config.Cfg.Slb.SvrName,
			TaskId:          "",
			TaskType:        tables.TaskTypeNormal,
			ParentAccountId: accountId,
			Action:          req.Action,
			RefOutpoint:     "",
			BlockNumber:     0,
			Outpoint:        common.OutPoint2String(hash.Hex(), 1),
			Timestamp:       time.Now().UnixNano() / 1e6,
			SmtStatus:       tables.SmtStatusWriteComplete,
			TxStatus:        tables.TxStatusPending,
			Retry:           0,
			CustomScripHash: "",
		}
		taskInfo.InitTaskId()
		if err := h.DbDao.CreateTask(&taskInfo); err != nil {
			log.Error("CreateTask err: ", err.Error())
		}
	}
	return nil
}

func (h *HttpHandle) doActionUpdateSubAccount(req *ReqTransactionSend, apiResp *api_code.ApiResp, resp *RespTransactionSend) error {
	var dataCache UpdateSubAccountCache
	if txStr, err := h.RC.GetSignTxCache(req.SignKey); err != nil {
		if err == redis.Nil {
			apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
		} else {
			apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
		}
		return fmt.Errorf("GetSignTxCache err: %s", err.Error())
	} else if err = json.Unmarshal([]byte(txStr), &dataCache); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
		return fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}
	log.Warn("UpdateSubAccountCache:", dataCache.Account, dataCache.SubAction)

	switch dataCache.SubAction {
	case common.SubActionCreate:
		if err := h.doSubActionCreate(dataCache, req, apiResp); err != nil {
			return fmt.Errorf("doSubActionCreate err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}
	case common.SubActionEdit:
		if err := h.doSubActionEdit(dataCache, req, apiResp); err != nil {
			return fmt.Errorf("doSubActionEdit err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}
	default:
		apiResp.ApiRespErr(api_code.ApiCodeNotExistConfirmAction, fmt.Sprintf("not exist sub action[%s]", dataCache.SubAction))
		return nil
	}
	return nil
}

func (h *HttpHandle) doSubActionEdit(dataCache UpdateSubAccountCache, req *ReqTransactionSend, apiResp *api_code.ApiResp) error {
	signAddress, subAcc, err := dataCache.EditCheck(h.DbDao, apiResp)
	if err != nil {
		return fmt.Errorf("EditCheck err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	signData := dataCache.GetEditSignData(h.DasCore.Daf(), subAcc, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if signData.SignMsg != dataCache.OldSignMsg {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "SignMsg diff")
		return nil
	}

	log.Warn("SubActionEdit:", signData.SignMsg, signAddress)

	signMsg := req.List[0].SignList[0].SignMsg

	if signMsg, err = doSignCheck(signData, signMsg, signAddress, apiResp); err != nil {
		return fmt.Errorf("doSignCheck err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// add record
	smtRecord := tables.TableSmtRecordInfo{
		Id:              0,
		SvrName:         config.Cfg.Slb.SvrName,
		AccountId:       subAcc.AccountId,
		Nonce:           subAcc.Nonce + 1,
		RecordType:      tables.RecordTypeDefault,
		TaskId:          "",
		Action:          common.DasActionUpdateSubAccount,
		ParentAccountId: subAcc.ParentAccountId,
		Account:         subAcc.Account,
		Content:         "",
		RegisterYears:   0,
		RegisterArgs:    "",
		EditKey:         dataCache.EditKey,
		Signature:       signMsg,
		EditArgs:        "",
		RenewYears:      0,
		EditRecords:     "",
		Timestamp:       time.Now().UnixNano() / 1e6,
		SubAction:       common.SubActionEdit,
		MintSignId:      "",
		ExpiredAt:       dataCache.ExpiredAt,
	}

	if err := dataCache.ConvertEditValue(h.DasCore.Daf(), subAcc, &smtRecord); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return err
	}

	if err := h.DbDao.CreateSmtRecordInfo(smtRecord); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "fail to create smt record")
		return fmt.Errorf("CreateSmtRecordInfo err:%s", err.Error())
	}
	return nil
}

func (h *HttpHandle) doSubActionCreate(dataCache UpdateSubAccountCache, req *ReqTransactionSend, apiResp *api_code.ApiResp) error {
	acc, err := h.DbDao.GetAccountInfoByAccountId(dataCache.ParentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "fail to search account")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
	}
	signData := dataCache.GetCreateSignData(&acc, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if signData.SignMsg != dataCache.OldSignMsg {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "SignMsg diff")
		return nil
	}

	signMsg := req.List[0].SignList[0].SignMsg

	if signMsg, err = doSignCheck(signData, signMsg, acc.Manager, apiResp); err != nil {
		return fmt.Errorf("doSignCheck err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	dataCache.MinSignInfo.Signature = signMsg

	if err := h.DbDao.CreateMinSignInfo(dataCache.MinSignInfo, dataCache.ListSmtRecord); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "fail to create mint sign info")
		return fmt.Errorf("CreateMinSignInfo err:%s", err.Error())
	}
	return nil
}

func doSignCheck(signData txbuilder.SignData, signMsg, signAddress string, apiResp *api_code.ApiResp) (string, error) {
	signOk := false
	var err error
	switch signData.SignType {
	case common.DasAlgorithmIdEth:
		signMsg = fixSignature(signMsg)
		signOk, err = sign.VerifyPersonalSignature(common.Hex2Bytes(signMsg), []byte(signData.SignMsg), signAddress)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "eth sign error")
			return "", fmt.Errorf("VerifyPersonalSignature err: %s", err.Error())
		}
	case common.DasAlgorithmIdTron:
		signMsg = fixSignature(signMsg)
		if signAddress, err = common.TronHexToBase58(signAddress); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "TronHexToBase58 error")
			return "", fmt.Errorf("TronHexToBase58 err: %s [%s]", err.Error(), signAddress)
		}
		signOk = sign.TronVerifySignature(true, common.Hex2Bytes(signMsg), []byte(signData.SignMsg), signAddress)
	case common.DasAlgorithmIdEd25519:
		signOk = sign.VerifyEd25519Signature(common.Hex2Bytes(signAddress), common.Hex2Bytes(signData.SignMsg), common.Hex2Bytes(signMsg))
	case common.DasAlgorithmIdDogeChain:
		signOk, err = sign.VerifyDogeSignature(common.Hex2Bytes(signMsg), []byte(signData.SignMsg), signAddress)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "VerifyDogeSignature error")
			return "", fmt.Errorf("VerifyDogeSignature err: %s [%s]", err.Error(), signAddress)
		}
	default:
		apiResp.ApiRespErr(api_code.ApiCodeNotExistSignType, fmt.Sprintf("not exist sign type[%d]", signData.SignType))
		return "", nil
	}

	if !signOk {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "res sign error")
	}
	return signMsg, nil
}

func fixSignature(signMsg string) string {
	if len(signMsg) >= 132 && signMsg[130:132] == "1b" {
		signMsg = signMsg[0:130] + "00" + signMsg[132:]
	}
	if len(signMsg) >= 132 && signMsg[130:132] == "1c" {
		signMsg = signMsg[0:130] + "01" + signMsg[132:]
	}
	return signMsg
}
