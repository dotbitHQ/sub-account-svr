package handle

import (
	"bytes"
	"crypto/md5"
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"encoding/binary"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ReqSubAccountEdit struct {
	core.ChainTypeAddress
	chainType common.ChainType
	address   string
	Account   string   `json:"account"`
	EditKey   string   `json:"edit_key"`
	EditValue EditInfo `json:"edit_value"`
}

type EditInfo struct {
	Owner   core.ChainTypeAddress `json:"owner"`
	Manager core.ChainTypeAddress `json:"manager"`
	Records []EditRecord          `json:"records"`
	//
	OwnerChainType   common.ChainType `json:"owner_chain_type"`
	OwnerAddress     string           `json:"owner_address"`
	ManagerChainType common.ChainType `json:"manager_chain_type"`
	ManagerAddress   string           `json:"manager_address"`
}

type EditRecord struct {
	Index int    `json:"index"`
	Key   string `json:"key"`
	Type  string `json:"type"`
	Label string `json:"label"`
	Value string `json:"value"`
	TTL   string `json:"ttl"`
}

type EditRecordList []EditRecord

func (e EditRecordList) Len() int { return len(e) }
func (e EditRecordList) Less(i, j int) bool {
	return e[i].Index < e[j].Index
}
func (e EditRecordList) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

type RespSubAccountEdit struct {
	SignInfoList
}

func (h *HttpHandle) SubAccountEdit(ctx *gin.Context) {
	var (
		funcName = "SubAccountEdit"
		clientIp = GetClientIp(ctx)
		req      ReqSubAccountEdit
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

	if err = h.doSubAccountEdit(&req, &apiResp); err != nil {
		log.Error("doSubAccountEdit err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountEdit(req *ReqSubAccountEdit, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountEdit
	resp.List = make([]SignInfo, 0)

	// check params
	h.CheckReqSubAccountEdit(req, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
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

	// check edit value
	var editCache EditSubAccountCache
	editCache.ChainType = req.chainType
	editCache.Address = req.address
	editCache.Account = req.Account
	editCache.EditKey = req.EditKey
	editCache.EditValue = req.EditValue
	_, subAcc, err := editCache.CheckEditValue(h.DbDao, apiResp)
	if err != nil {
		return fmt.Errorf("CheckEditValue err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	log.Warn("EditSubAccountCache:", toolib.JsonString(&editCache))

	if _, ok := config.Cfg.SuspendMap[subAcc.ParentAccountId]; ok {
		apiResp.ApiRespErr(api_code.ApiCodeSuspendOperation, "suspend operation")
		return nil
	}

	// check nonce
	if record, err := h.DbDao.GetLatestNonceSmtRecordByAccountId(subAcc.AccountId, tables.RecordTypeDefault); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return fmt.Errorf("GetLatestNonceSmtRecordByAccountId: %s", err.Error())
	} else if record.Nonce > subAcc.Nonce {
		apiResp.ApiRespErr(api_code.ApiCodeRecordDoing, "task not completed")
		return nil
	}

	// cache sign info
	signData := editCache.GetSignData(h.DasCore.Daf(), subAcc, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	editCache.OldSignMsg = signData.SignMsg // for check after user sign

	// cache
	signKey := editCache.CacheKey()
	cacheStr := toolib.JsonString(&editCache)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	// resp
	resp.Action = common.DasActionEditSubAccount
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		//SignKey: "",
		SignList: []txbuilder.SignData{
			signData,
		},
	})

	apiResp.ApiRespOK(resp)
	return nil
}

// ===========

func (h *HttpHandle) CheckReqSubAccountEdit(r *ReqSubAccountEdit, apiResp *api_code.ApiResp) {
	// check params
	addrHex, err := r.FormatChainTypeAddress(config.Cfg.Server.Net, true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
		return
	}
	r.chainType, r.address = addrHex.ChainType, addrHex.AddressHex

	// check edit value
	switch r.EditKey {
	case common.EditKeyOwner:
		addrHex, err = r.EditValue.Owner.FormatChainTypeAddress(config.Cfg.Server.Net, true)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
			return
		}
		r.EditValue.OwnerChainType, r.EditValue.OwnerAddress = addrHex.ChainType, addrHex.AddressHex
	case common.EditKeyManager:
		addrHex, err = r.EditValue.Manager.FormatChainTypeAddress(config.Cfg.Server.Net, true)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
			return
		}
		r.EditValue.ManagerChainType, r.EditValue.ManagerAddress = addrHex.ChainType, addrHex.AddressHex
	case common.EditKeyRecords:
		builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsRecordNamespace)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return
		}
		log.Info("ConfigCellRecordKeys:", builder.ConfigCellRecordKeys)
		var mapRecordKey = make(map[string]struct{})
		for _, v := range builder.ConfigCellRecordKeys {
			mapRecordKey[v] = struct{}{}
		}
		for i, v := range r.EditValue.Records {
			r.EditValue.Records[i].Index = i
			record := fmt.Sprintf("%s.%s", v.Type, v.Key)
			if v.Type == "custom_key" { // (^[0-9a-z_]+$)
				if ok, _ := regexp.MatchString("^[0-9a-z_]+$", v.Key); !ok {
					apiResp.ApiRespErr(api_code.ApiCodeRecordInvalid, fmt.Sprintf("record [%s] is invalid", record))
					return
				}
			} else if v.Type == "address" {
				if ok, _ := regexp.MatchString("^(0|[1-9][0-9]*)$", v.Key); !ok {
					apiResp.ApiRespErr(api_code.ApiCodeRecordInvalid, fmt.Sprintf("record [%s] is invalid", record))
					return
				}
			} else if _, ok := mapRecordKey[record]; !ok {
				apiResp.ApiRespErr(api_code.ApiCodeRecordInvalid, fmt.Sprintf("record [%s] is invalid", record))
				return
			}
		}
	default:
		apiResp.ApiRespErr(api_code.ApiCodeNotExistEditKey, fmt.Sprintf("not exist edit key [%s]", r.EditKey))
		return
	}
}

// ===========
type EditSubAccountCache struct {
	ChainType  common.ChainType `json:"chain_type"`
	Address    string           `json:"address"`
	Account    string           `json:"account"`
	EditKey    string           `json:"edit_key"`
	EditValue  EditInfo         `json:"edit_value"`
	OldSignMsg string           `json:"old_sign_msg"`
}

func (e *EditSubAccountCache) CacheKey() string {
	key := fmt.Sprintf("%s%d%s%s%d", common.DasActionEditSubAccount, e.ChainType, e.Address, e.EditKey, time.Now().UnixNano())
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}

func (e *EditSubAccountCache) GetSignData(daf *core.DasAddressFormat, subAcc *tables.TableAccountInfo, apiResp *api_code.ApiResp) (signData txbuilder.SignData) {
	subAccountId := subAcc.AccountId
	data := common.Hex2Bytes(subAccountId)
	data = append(data, []byte(e.EditKey)...)
	log.Info("GetSignData:", e.EditKey)
	switch e.EditKey {
	case common.EditKeyOwner:
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: e.EditValue.OwnerChainType.ToDasAlgorithmId(true),
			AddressHex:     e.EditValue.OwnerAddress,
			IsMulti:        false,
			ChainType:      e.EditValue.OwnerChainType,
		}
		args, err := daf.HexToArgs(ownerHex, ownerHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("HexToArgs err: %s", err.Error()))
			return
		}
		data = append(data, args...)
		log.Info("GetSignData:", common.Bytes2Hex(args))
		signData.SignType = subAcc.OwnerAlgorithmId
	case common.EditKeyManager:
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: subAcc.OwnerChainType.ToDasAlgorithmId(true),
			AddressHex:     subAcc.Owner,
			IsMulti:        false,
			ChainType:      subAcc.OwnerChainType,
		}
		managerHex := core.DasAddressHex{
			DasAlgorithmId: e.EditValue.ManagerChainType.ToDasAlgorithmId(true),
			AddressHex:     e.EditValue.ManagerAddress,
			IsMulti:        false,
			ChainType:      e.EditValue.ManagerChainType,
		}
		args, err := daf.HexToArgs(ownerHex, managerHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("HexToArgs err: %s", err.Error()))
			return
		}
		data = append(data, args...)
		log.Info("GetSignData:", common.Bytes2Hex(args))
		signData.SignType = subAcc.OwnerAlgorithmId
	case common.EditKeyRecords:
		list := e.FormatRecords()
		records := witness.ConvertToCellRecords(list)
		bys := records.AsSlice()
		log.Info("GetSignData:", common.Bytes2Hex(bys))
		data = append(data, bys...)
		signData.SignType = subAcc.ManagerAlgorithmId
	default:
		apiResp.ApiRespErr(api_code.ApiCodeNotExistEditKey, fmt.Sprintf("not exist edit key [%s]", e.EditKey))
		return
	}
	// nonce
	nonce := subAcc.Nonce
	nonceByte := bytes.NewBuffer([]byte{})
	_ = binary.Write(nonceByte, binary.LittleEndian, nonce)
	data = append(data, nonceByte.Bytes()...)

	// sig msg
	if signData.SignType == common.DasAlgorithmIdEth712 {
		signData.SignType = common.DasAlgorithmIdEth
	}
	log.Info("GetSignData:", common.Bytes2Hex(data))
	bys, _ := blake2b.Blake256(data)
	log.Info("GetSignData:", common.Bytes2Hex(bys))

	//if signData.SignType == common.DasAlgorithmIdTron {
	//	signData.SignMsg = common.Bytes2Hex([]byte("from did: "))[2:] + common.Bytes2Hex(bys)[2:]
	//} else {
	//	signData.SignMsg = "from did: " + common.Bytes2Hex(bys)[2:]
	//}

	signData.SignMsg = common.Bytes2Hex([]byte("from did: "))[2:] + common.Bytes2Hex(bys)[2:]
	return
}

func (e *EditSubAccountCache) CheckEditValue(db *dao.DbDao, apiResp *api_code.ApiResp) (string, *tables.TableAccountInfo, error) {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(e.Account))
	signAddress := ""
	// check account
	subAcc, err := db.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account")
		return "", nil, fmt.Errorf("GetAccountInfoByAccountId: %s", err.Error())
	} else if subAcc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return "", nil, nil
	} else if subAcc.Status != tables.AccountStatusNormal {
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "account status is not normal")
		return "", nil, nil
	} else if subAcc.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, "account expired")
		return "", nil, nil
	} else if subAcc.ParentAccountId == "" {
		apiResp.ApiRespErr(api_code.ApiCodeNotSubAccount, fmt.Sprintf("%s not a sub account", e.Account))
		return "", nil, nil
	}

	// check Permission
	if e.EditKey == common.EditKeyOwner {
		if e.ChainType != subAcc.OwnerChainType || !strings.EqualFold(e.Address, subAcc.Owner) {
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
			return "", nil, nil
		} else if e.EditValue.OwnerChainType == subAcc.OwnerChainType && strings.EqualFold(e.EditValue.OwnerAddress, subAcc.Owner) {
			apiResp.ApiRespErr(api_code.ApiCodeSameLock, "same address")
			return "", nil, nil
		}
		signAddress = subAcc.Owner
	} else if e.EditKey == common.EditKeyManager {
		if e.ChainType != subAcc.OwnerChainType || !strings.EqualFold(e.Address, subAcc.Owner) {
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
			return "", nil, nil
		} else if e.EditValue.ManagerChainType == subAcc.ManagerChainType && strings.EqualFold(e.EditValue.ManagerAddress, subAcc.Manager) {
			apiResp.ApiRespErr(api_code.ApiCodeSameLock, "same address")
			return "", nil, nil
		}
		signAddress = subAcc.Owner
	} else if e.EditKey == common.EditKeyRecords {
		if e.ChainType != subAcc.ManagerChainType || !strings.EqualFold(e.Address, subAcc.Manager) {
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
			return "", nil, nil
		}
		signAddress = subAcc.Manager
		// max records
		records := e.FormatRecords()
		wi := witness.ConvertToCellRecords(records)
		if wi.TotalSize() > 4800 {
			apiResp.ApiRespErr(api_code.ApiCodeRecordsTotalLengthExceeded, fmt.Sprintf("records len exceeded, current: [%d]", wi.TotalSize()))
			return "", nil, nil
		}
		log.Info("wi.TotalSize:", wi.TotalSize())
	} else {
		apiResp.ApiRespErr(api_code.ApiCodeNotExistEditKey, fmt.Sprintf("not exist edit key [%s]", e.EditKey))
		return "", nil, nil
	}
	return signAddress, &subAcc, nil
}

func (e *EditSubAccountCache) InitRecord(daf *core.DasAddressFormat, subAcc *tables.TableAccountInfo, record *tables.TableSmtRecordInfo) error {
	switch e.EditKey {
	case common.EditKeyOwner:
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: e.EditValue.OwnerChainType.ToDasAlgorithmId(true),
			AddressHex:     e.EditValue.OwnerAddress,
			IsMulti:        false,
			ChainType:      e.EditValue.OwnerChainType,
		}
		args, err := daf.HexToArgs(ownerHex, ownerHex)
		if err != nil {
			return fmt.Errorf("HexToArgs err: %s", err.Error())
		}
		record.EditArgs = common.Bytes2Hex(args)
	case common.EditKeyManager:
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: subAcc.OwnerChainType.ToDasAlgorithmId(true),
			AddressHex:     subAcc.Owner,
			IsMulti:        false,
			ChainType:      subAcc.OwnerChainType,
		}
		managerHex := core.DasAddressHex{
			DasAlgorithmId: e.EditValue.ManagerChainType.ToDasAlgorithmId(true),
			AddressHex:     e.EditValue.ManagerAddress,
			IsMulti:        false,
			ChainType:      e.EditValue.ManagerChainType,
		}
		args, err := daf.HexToArgs(ownerHex, managerHex)
		if err != nil {
			return fmt.Errorf("HexToArgs err: %s", err.Error())
		}
		record.EditArgs = common.Bytes2Hex(args)
	case common.EditKeyRecords:
		records := e.FormatRecords()
		record.EditRecords = toolib.JsonString(records)
	default:
		return fmt.Errorf("not exist edit key [%s]", e.EditKey)
	}
	return nil
}

func (e *EditSubAccountCache) FormatRecords() []witness.Record {
	sort.Sort(EditRecordList(e.EditValue.Records))
	records := make([]witness.Record, 0)
	for _, v := range e.EditValue.Records {
		ttl, _ := strconv.ParseInt(v.TTL, 10, 64)
		if ttl <= 0 {
			ttl = 300
		}
		tmp := witness.Record{
			Key:   v.Key,
			Type:  v.Type,
			Label: v.Label,
			Value: v.Value,
			TTL:   uint32(ttl),
		}
		records = append(records, tmp)
	}
	return records
}
