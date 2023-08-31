package handle

import (
	"das_sub_account/config"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
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

	// check update
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}

	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	if err := h.doEditSignMsg(req, apiResp); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("doEditSignMsg err: %s", err.Error())
	}

	switch req.Action {
	case common.DasActionEnableSubAccount, common.DasActionConfigSubAccountCustomScript, common.DasActionConfigSubAccount:
		if err := h.doActionNormal(req, apiResp, &resp); err != nil {
			return fmt.Errorf("doActionNormal err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}
	case common.DasActionCreateApproval, common.DasActionDelayApproval,
		common.DasActionRevokeApproval, common.DasActionFulfillApproval:
		if err := h.doApproval(req, apiResp, &resp); err != nil {
			return fmt.Errorf("doApproval err: %s", err.Error())
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
		address := ""
		signType := req.List[0].SignList[0].SignType
		signature := req.List[0].SignList[0].SignMsg
		if signType == common.DasAlgorithmIdWebauthn {
			address = req.SignAddress
		} else {
			address = res.AddressHex
		}
		verifyRes, signature, err := api_code.VerifySignature(signType, signMsg, signature, address)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "VerifySignature err: "+err.Error())
			return fmt.Errorf("VerifySignature err: %s", err.Error())
		}
		if !verifyRes {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "res sign error")
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
		address := ""
		signType := req.List[0].SignList[0].SignType
		signature := req.List[0].SignList[0].SignMsg
		if signType == common.DasAlgorithmIdWebauthn {
			address = req.SignAddress
		} else {
			address = res.AddressHex
		}
		verifyRes, signature, err := api_code.VerifySignature(signType, signMsg, signature, address)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "VerifySignature err: "+err.Error())
			return fmt.Errorf("VerifySignature err: %s", err.Error())
		}
		if !verifyRes {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "res sign error")
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
			MintSuccessPage: data.MintSuccessPage,
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

func (h *HttpHandle) doApproval(req *ReqTransactionSend, apiResp *api_code.ApiResp, resp *RespTransactionSend) error {
	var sic SignInfoCache
	txStr, err := h.RC.GetSignTxCache(req.SignKey)
	if err != nil {
		if err == redis.Nil {
			apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
		} else {
			apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
		}
		return fmt.Errorf("GetSignTxCache err: %s", err.Error())
	}

	if err := json.Unmarshal([]byte(txStr), &sic); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
		return fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, sic.BuilderTx)
	if len(req.List) > 0 {
		if err := txBuilder.AddSignatureForTx(req.List[0].SignList); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "add signature fail")
			return fmt.Errorf("AddSignatureForTx err: %s", err.Error())
		}
	} else {
		if err := txBuilder.AddSignatureForTx(req.SignList); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "add signature fail")
			return fmt.Errorf("AddSignatureForTx err: %s", err.Error())
		}
	}

	hash, err := txBuilder.SendTransaction()
	if err != nil {
		return doSendTransactionError(err, apiResp)
	}
	h.DasCache.AddCellInputByAction("", sic.BuilderTx.Transaction.Inputs)
	resp.HashList = append(resp.HashList, hash.Hex())
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
	log.Info("UpdateSubAccountCache:", dataCache.Account, dataCache.SubAction)

	switch dataCache.SubAction {
	case common.SubActionCreate, common.SubActionRenew:
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
	case common.DasActionCreateApproval, common.DasActionDelayApproval,
		common.DasActionRevokeApproval, common.DasActionFulfillApproval:
		if err := h.doSubActionApproval(dataCache, req, apiResp); err != nil {
			return fmt.Errorf("doSubActionApproval err: %s", err.Error())
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
	loginAddress, subAcc, err := dataCache.EditCheck(h.DbDao, apiResp)
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

	log.Warn("SubActionEdit:", signData.SignMsg, loginAddress)

	signature := req.List[0].SignList[0].SignMsg

	address := ""
	if signData.SignType == common.DasAlgorithmIdWebauthn {
		address = req.SignAddress
	} else {
		address = loginAddress
	}
	verifyRes, signature, err := api_code.VerifySignature(signData.SignType, signData.SignMsg, signature, address)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "VerifySignature err: "+err.Error())
		return fmt.Errorf("VerifySignature err: %s", err.Error())
	}
	if !verifyRes {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "res sign error")
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
		Signature:       signature,
		LoginChainType:  dataCache.ChainType,
		LoginAddress:    dataCache.Address,
		SignAddress:     req.SignAddress,
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
	signData := dataCache.GetCreateSignData(acc.ManagerAlgorithmId, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if signData.SignMsg != dataCache.OldSignMsg {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "SignMsg diff")
		return nil
	}

	signature := req.List[0].SignList[0].SignMsg

	address := ""
	if signData.SignType == common.DasAlgorithmIdWebauthn {
		address = req.SignAddress
	} else {
		address = acc.Manager
	}
	verifyRes, signature, err := api_code.VerifySignature(signData.SignType, signData.SignMsg, signature, address)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "VerifySignature err: "+err.Error())
		return fmt.Errorf("VerifySignature err: %s", err.Error())
	}
	if !verifyRes {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "res sign error")
		return nil
	}

	dataCache.MinSignInfo.Signature = signature
	for i, _ := range dataCache.ListSmtRecord {
		dataCache.ListSmtRecord[i].LoginChainType = dataCache.ChainType //login chain_type
		dataCache.ListSmtRecord[i].LoginAddress = dataCache.Address     //login addr
		dataCache.ListSmtRecord[i].SignAddress = req.SignAddress        //sign addr
		dataCache.ListSmtRecord[i].Signature = signature
	}
	if err := h.DbDao.CreateMinSignInfo(*dataCache.MinSignInfo, dataCache.ListSmtRecord); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "fail to create mint sign info")
		return fmt.Errorf("CreateMinSignInfo err:%s", err.Error())
	}
	return nil
}

func (h *HttpHandle) doSubActionApproval(dataCache UpdateSubAccountCache, req *ReqTransactionSend, apiResp *api_code.ApiResp) error {
	acc, err := h.DbDao.GetAccountInfoByAccountId(dataCache.AccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "fail to search account")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
	}

	var approvalInfo tables.ApprovalInfo
	if dataCache.SubAction != common.SubActionCreateApproval {
		approvalInfo, err = h.DbDao.GetAccountPendingApproval(dataCache.AccountId)
		if err != nil {
			return err
		}
		if approvalInfo.ID == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeAccountApprovalNotExist, "account approval not exist")
			return fmt.Errorf("account approval not exist")
		}
	}

	var signature string
	if dataCache.SubAction != common.SubActionFullfillApproval ||
		uint64(time.Now().Unix()) < approvalInfo.SealedUntil {

		addr := acc.Owner
		if dataCache.SubAction == common.SubActionRevokeApproval {
			addr = approvalInfo.Platform
		}
		signature = req.List[0].SignList[0].SignMsg
		if signature, err = doSignCheck(dataCache.SignData, signature, addr, addr, apiResp); err != nil {
			return fmt.Errorf("doSignCheck err: %s", err.Error())
		} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
			return nil
		}
	}

	for i := range dataCache.ListSmtRecord {
		dataCache.ListSmtRecord[i].LoginChainType = dataCache.ChainType //login chain_type
		dataCache.ListSmtRecord[i].LoginAddress = dataCache.Address     //login addr
		dataCache.ListSmtRecord[i].SignAddress = req.SignAddress        //sign addr
		dataCache.ListSmtRecord[i].Signature = signature                //sign msg
	}
	if err := h.DbDao.CreateSmtRecordList(dataCache.ListSmtRecord); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "fail to create smt record info")
		return fmt.Errorf("CreateSmtRecordList err:%s", err.Error())
	}
	return nil
}

func (h *HttpHandle) doEditSignMsg(req *ReqTransactionSend, apiResp *api_code.ApiResp) error {
	hasWebAuthn := false
	for _, signInfo := range req.List {
		for _, v := range signInfo.SignList {
			if v.SignType == common.DasAlgorithmIdWebauthn {
				hasWebAuthn = true
				break
			}
		}
	}
	if !hasWebAuthn {
		return nil
	}

	txAddr := ""
	switch req.Action {
	case common.DasActionEnableSubAccount, common.DasActionConfigSubAccountCustomScript, common.DasActionConfigSubAccount:
		var sic SignInfoCache
		txStr, err := h.RC.GetSignTxCache(req.SignKey)
		if err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			return fmt.Errorf("GetSignTxCache err: %s", err.Error())
		}
		if err = json.Unmarshal([]byte(txStr), &sic); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			return fmt.Errorf("json.Unmarshal err: %s", err.Error())
		}
		txAddr = sic.Address
	case common.DasActionUpdateSubAccount:
		var dataCache UpdateSubAccountCache
		txStr, err := h.RC.GetSignTxCache(req.SignKey)
		if err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			return fmt.Errorf("GetSignTxCache err: %s", err.Error())
		}
		if err = json.Unmarshal([]byte(txStr), &dataCache); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			return fmt.Errorf("json.Unmarshal err: %s", err.Error())
		}
		txAddr = dataCache.Address
	case ActionCurrencyUpdate, ActionMintConfigUpdate:
		chainTypeAddress := &core.ChainTypeAddress{}
		txStr, err := h.RC.GetSignTxCache(req.SignKey)
		if err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			return fmt.Errorf("GetSignTxCache err: %s", err.Error())
		}
		if err = json.Unmarshal([]byte(txStr), &chainTypeAddress); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			return fmt.Errorf("json.Unmarshal err: %s", err.Error())
		}
		txAddrAddressHex, err := h.DasCore.Daf().NormalToHex(core.DasAddressNormal{
			ChainType:     common.ChainTypeWebauthn,
			AddressNormal: chainTypeAddress.KeyInfo.Key,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "chainTypeAddress.KeyInfo.Key  NormalToHex err")
			return err
		}
		txAddr = txAddrAddressHex.AddressHex
	}

	loginAddrHex := core.DasAddressHex{
		DasAlgorithmId:    common.DasAlgorithmIdWebauthn,
		DasSubAlgorithmId: common.DasWebauthnSubAlgorithmIdES256,
		AddressHex:        txAddr,
		AddressPayload:    common.Hex2Bytes(txAddr),
		ChainType:         common.ChainTypeWebauthn,
	}

	signAddressHex, err := h.DasCore.Daf().NormalToHex(core.DasAddressNormal{
		ChainType:     common.ChainTypeWebauthn,
		AddressNormal: req.SignAddress,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "sign address NormalToHex err")
		return err
	}
	req.SignAddress = signAddressHex.AddressHex
	log.Info("-----", loginAddrHex.AddressHex, "--", signAddressHex.AddressHex)
	idx, err := h.DasCore.GetIdxOfKeylist(loginAddrHex, signAddressHex)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "GetIdxOfKeylist err: "+err.Error())
		return fmt.Errorf("GetIdxOfKeylist err: %s", err.Error())
	}
	if idx == -1 {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return fmt.Errorf("permission denied")
	}

	for i, signList := range req.List {
		for j, _ := range signList.SignList {
			if req.List[i].SignList[j].SignType == common.DasAlgorithmIdWebauthn {
				h.DasCore.AddPkIndexForSignMsg(&req.List[i].SignList[j].SignMsg, idx)
			}
		}
	}
	return nil
}

func beforeSignCheck(dc *core.DasCore, loginAddress, signAddress core.DasAddressHex) error {
	res, err := dc.GetIdxOfKeylist(loginAddress, signAddress)
	if err != nil {
		return fmt.Errorf("GetIdxOfKeylist err: %s", err.Error())
	}
	if res == -1 {
		return fmt.Errorf("permission denied")
	}
	return nil
}

func doSignCheck(signData txbuilder.SignData, signMsg, loginAddress, signAddress string, apiResp *api_code.ApiResp) (string, error) {
	signOk := false
	var err error
	switch signData.SignType {
	case common.DasAlgorithmIdEth:
		signMsg = fixSignature(signMsg)
		signOk, err = sign.VerifyPersonalSignature(common.Hex2Bytes(signMsg), []byte(signData.SignMsg), loginAddress)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "eth sign error")
			return "", fmt.Errorf("VerifyPersonalSignature err: %s", err.Error())
		}
	case common.DasAlgorithmIdTron:
		signMsg = fixSignature(signMsg)
		if loginAddress, err = common.TronHexToBase58(loginAddress); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "TronHexToBase58 error")
			return "", fmt.Errorf("TronHexToBase58 err: %s [%s]", err.Error(), signAddress)
		}
		signOk = sign.TronVerifySignature(true, common.Hex2Bytes(signMsg), []byte(signData.SignMsg), loginAddress)
	case common.DasAlgorithmIdEd25519:
		signOk = sign.VerifyEd25519Signature(common.Hex2Bytes(loginAddress), common.Hex2Bytes(signData.SignMsg), common.Hex2Bytes(signMsg))
	case common.DasAlgorithmIdDogeChain:
		signOk, err = sign.VerifyDogeSignature(common.Hex2Bytes(signMsg), []byte(signData.SignMsg), loginAddress)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "VerifyDogeSignature error")
			return "", fmt.Errorf("VerifyDogeSignature err: %s [%s]", err.Error(), signAddress)
		}
	case common.DasAlgorithmIdWebauthn:
		//no need to verify if signAddr is in loginaddr`s keylist
		signOk, err = sign.VerifyWebauthnSignature([]byte(signData.SignMsg), common.Hex2Bytes(signMsg), signAddress[20:])
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeSignError, "VerifyWebauthnSignature error")
			return "", fmt.Errorf("VerifyWebauthnSignature err: %s [%s]", err.Error(), signAddress)
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
