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
	"encoding/hex"
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

type RespSubAccountEdit struct {
	SignInfoList
}

func (h *HttpHandle) checkReqSubAccountEdit(r *ReqSubAccountEdit, apiResp *api_code.ApiResp) {
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
					if _, ok2 := mapRecordKey[record]; !ok2 {
						apiResp.ApiRespErr(api_code.ApiCodeRecordInvalid, fmt.Sprintf("record [%s] is invalid", record))
						return
					}
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

func (h *HttpHandle) SubAccountEditNew(ctx *gin.Context) {
	var (
		funcName               = "SubAccountEditNew"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSubAccountEdit
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

	if err = h.doSubAccountEditNew(&req, &apiResp); err != nil {
		log.Error("doSubAccountEditNew err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}
func (h *HttpHandle) doSubAccountEditNew(req *ReqSubAccountEdit, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountEdit
	resp.List = make([]SignInfo, 0)

	// check params
	h.checkReqSubAccountEdit(req, apiResp)
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
	dataCache := UpdateSubAccountCache{
		ParentAccountId: "",
		Account:         req.Account,
		ChainType:       req.chainType,
		Address:         req.address,
		SubAction:       common.SubActionEdit,
		EditKey:         req.EditKey,
		EditValue:       req.EditValue,
		OldSignMsg:      "",
		ExpiredAt:       0,
	}
	_, subAcc, err := dataCache.EditCheck(h.DbDao, apiResp)
	if err != nil {
		return fmt.Errorf("EditCheck err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	dataCache.ParentAccountId = subAcc.ParentAccountId
	log.Info("doSubAccountEditNew:", toolib.JsonString(&dataCache))

	if _, ok := config.Cfg.SuspendMap[subAcc.ParentAccountId]; ok {
		apiResp.ApiRespErr(api_code.ApiCodeSuspendOperation, "suspend operation")
		return nil
	}
	// ExpiredAt
	acc, err := h.DbDao.GetAccountInfoByAccountId(subAcc.ParentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "get account err")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
	} else if acc.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountExpired, "Parent account expired")
		return nil
	}
	dataCache.ExpiredAt = uint64(time.Now().Add(time.Hour * 24 * 7).Unix())
	if dataCache.ExpiredAt > acc.ExpiredAt {
		dataCache.ExpiredAt = acc.ExpiredAt
	}
	if dataCache.ExpiredAt > subAcc.ExpiredAt {
		dataCache.ExpiredAt = subAcc.ExpiredAt
	}

	// sign info
	signData := dataCache.GetEditSignData(h.DasCore.Daf(), subAcc, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	dataCache.OldSignMsg = signData.SignMsg // for check after user sign

	// cache
	signKey := dataCache.CacheKey()
	cacheStr := toolib.JsonString(&dataCache)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	// resp
	resp.Action = common.DasActionUpdateSubAccount
	resp.SubAction = common.SubActionEdit
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

// === UpdateSubAccount ===
type UpdateSubAccountCache struct {
	ParentAccountId string           `json:"parent_account_id"`
	Account         string           `json:"account"`
	ChainType       common.ChainType `json:"chain_type"`
	Address         string           `json:"address"`
	SubAction       common.SubAction `json:"sub_action"`
	EditKey         common.EditKey   `json:"edit_key"`
	EditValue       EditInfo         `json:"edit_value"`
	ExpiredAt       uint64           `json:"expired_at"`

	OldSignMsg    string                      `json:"old_sign_msg"`
	MinSignInfo   tables.TableMintSignInfo    `json:"min_sign_info"`
	ListSmtRecord []tables.TableSmtRecordInfo `json:"list_smt_record"`
}

func (u *UpdateSubAccountCache) CacheKey() string {
	key := fmt.Sprintf("%s%d%s%s%s%s%s%d", common.DasActionUpdateSubAccount, u.ChainType, u.Address, u.ParentAccountId, u.SubAction, u.Account, u.EditKey, time.Now().UnixNano())
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}

func (u *UpdateSubAccountCache) EditCheck(db *dao.DbDao, apiResp *api_code.ApiResp) (string, *tables.TableAccountInfo, error) {
	subAccId := common.Bytes2Hex(common.GetAccountIdByAccount(u.Account))
	subAcc, err := db.GetAccountInfoByAccountId(subAccId)
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
		apiResp.ApiRespErr(api_code.ApiCodeNotSubAccount, fmt.Sprintf("%s not a sub account", u.Account))
		return "", nil, nil
	}

	// check Permission
	signAddress := ""
	if u.EditKey == common.EditKeyOwner {
		if u.ChainType != subAcc.OwnerChainType || !strings.EqualFold(u.Address, subAcc.Owner) {
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "owner permission denied")
			return "", nil, nil
		} else if u.EditValue.OwnerChainType == subAcc.OwnerChainType && strings.EqualFold(u.EditValue.OwnerAddress, subAcc.Owner) {
			apiResp.ApiRespErr(api_code.ApiCodeSameLock, "same owner address")
			return "", nil, nil
		}
		signAddress = subAcc.Owner
	} else if u.EditKey == common.EditKeyManager {
		if u.ChainType != subAcc.OwnerChainType || !strings.EqualFold(u.Address, subAcc.Owner) {
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "owner permission denied")
			return "", nil, nil
		} else if u.EditValue.ManagerChainType == subAcc.ManagerChainType && strings.EqualFold(u.EditValue.ManagerAddress, subAcc.Manager) {
			apiResp.ApiRespErr(api_code.ApiCodeSameLock, "same manager address")
			return "", nil, nil
		}
		signAddress = subAcc.Owner
	} else if u.EditKey == common.EditKeyRecords {
		if u.ChainType != subAcc.ManagerChainType || !strings.EqualFold(u.Address, subAcc.Manager) {
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "manager permission denied")
			return "", nil, nil
		}
		signAddress = subAcc.Manager
		// max records
		records := u.FormatRecords()
		wi := witness.ConvertToCellRecords(records)
		if wi.TotalSize() > 4800 {
			apiResp.ApiRespErr(api_code.ApiCodeRecordsTotalLengthExceeded, fmt.Sprintf("records len exceeded, current: [%d]", wi.TotalSize()))
			return "", nil, nil
		}
		log.Warn("wi.TotalSize:", wi.TotalSize())
	} else {
		apiResp.ApiRespErr(api_code.ApiCodeNotExistEditKey, fmt.Sprintf("not exist edit key [%s]", u.EditKey))
		return "", nil, nil
	}

	// check nonce
	if record, err := db.GetLatestNonceSmtRecordByAccountId(subAcc.AccountId, tables.RecordTypeDefault); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return "", nil, fmt.Errorf("GetLatestNonceSmtRecordByAccountId: %s", err.Error())
	} else if record.Nonce > subAcc.Nonce {
		apiResp.ApiRespErr(api_code.ApiCodeRecordDoing, "task not completed")
		return "", nil, nil
	}

	return signAddress, &subAcc, nil
}

func (u *UpdateSubAccountCache) FormatRecords() []witness.Record {
	sort.Sort(EditRecordList(u.EditValue.Records))
	records := make([]witness.Record, 0)
	for _, v := range u.EditValue.Records {
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

func (u *UpdateSubAccountCache) GetEditSignData(daf *core.DasAddressFormat, subAcc *tables.TableAccountInfo, apiResp *api_code.ApiResp) (signData txbuilder.SignData) {
	subAccountId := subAcc.AccountId
	data := common.Hex2Bytes(subAccountId)
	data = append(data, []byte(u.EditKey)...)

	switch u.EditKey {
	case common.EditKeyOwner:
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: u.EditValue.OwnerChainType.ToDasAlgorithmId(true),
			AddressHex:     u.EditValue.OwnerAddress,
			IsMulti:        false,
			ChainType:      u.EditValue.OwnerChainType,
		}
		args, err := daf.HexToArgs(ownerHex, ownerHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("HexToArgs err: %s", err.Error()))
			return
		}
		data = append(data, args...)
		signData.SignType = subAcc.OwnerAlgorithmId
	case common.EditKeyManager:
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: subAcc.OwnerChainType.ToDasAlgorithmId(true),
			AddressHex:     subAcc.Owner,
			IsMulti:        false,
			ChainType:      subAcc.OwnerChainType,
		}
		managerHex := core.DasAddressHex{
			DasAlgorithmId: u.EditValue.ManagerChainType.ToDasAlgorithmId(true),
			AddressHex:     u.EditValue.ManagerAddress,
			IsMulti:        false,
			ChainType:      u.EditValue.ManagerChainType,
		}
		args, err := daf.HexToArgs(ownerHex, managerHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("HexToArgs err: %s", err.Error()))
			return
		}
		data = append(data, args...)
		signData.SignType = subAcc.OwnerAlgorithmId
	case common.EditKeyRecords:
		list := u.FormatRecords()
		records := witness.ConvertToCellRecords(list)
		bys := records.AsSlice()
		data = append(data, bys...)
		signData.SignType = subAcc.ManagerAlgorithmId
	default:
		apiResp.ApiRespErr(api_code.ApiCodeNotExistEditKey, fmt.Sprintf("not exist edit key [%s]", u.EditKey))
		return
	}

	// nonce
	nonce := subAcc.Nonce
	nonceByte := bytes.NewBuffer([]byte{})
	_ = binary.Write(nonceByte, binary.LittleEndian, nonce)
	data = append(data, nonceByte.Bytes()...)

	// sign_expired_at
	expiredAtBys := bytes.NewBuffer([]byte{})
	if err := binary.Write(expiredAtBys, binary.LittleEndian, u.ExpiredAt); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("binary.Write err: %s", err.Error()))
		return
	}
	data = append(data, expiredAtBys.Bytes()...)

	// sig msg
	if signData.SignType == common.DasAlgorithmIdEth712 {
		signData.SignType = common.DasAlgorithmIdEth
	}

	bys, _ := blake2b.Blake256(data)
	signData.SignMsg = common.PersonSignPrefix + hex.EncodeToString(bys)
	log.Info("GetEditSignData:", u.ExpiredAt, signData.SignMsg)
	return
}

func (u *UpdateSubAccountCache) GetCreateSignData(acc *tables.TableAccountInfo, apiResp *api_code.ApiResp) (signData txbuilder.SignData) {
	// ExpiredAt + SmtRoot

	expiredAtBys := bytes.NewBuffer([]byte{})
	if err := binary.Write(expiredAtBys, binary.LittleEndian, u.MinSignInfo.ExpiredAt); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("binary.Write err: %s", err.Error()))
		return
	}
	data := expiredAtBys.Bytes()

	data = append(data, common.Hex2Bytes(u.MinSignInfo.SmtRoot)...)

	bys, err := blake2b.Blake256(data)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("blake2b.Blake256 err: %s", err.Error()))
		return
	}
	signData.SignMsg = common.PersonSignPrefix + hex.EncodeToString(bys)
	log.Info("GetCreateSignData:", signData.SignMsg, u.MinSignInfo.ExpiredAt, u.MinSignInfo.SmtRoot)
	// sig msg
	signData.SignType = acc.ManagerAlgorithmId
	if signData.SignType == common.DasAlgorithmIdEth712 {
		signData.SignType = common.DasAlgorithmIdEth
	}

	return
}

// === Edit Value ===

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

func (u *UpdateSubAccountCache) ConvertEditValue(daf *core.DasAddressFormat, subAcc *tables.TableAccountInfo, record *tables.TableSmtRecordInfo) error {
	switch u.EditKey {
	case common.EditKeyOwner:
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: u.EditValue.OwnerChainType.ToDasAlgorithmId(true),
			AddressHex:     u.EditValue.OwnerAddress,
			IsMulti:        false,
			ChainType:      u.EditValue.OwnerChainType,
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
			DasAlgorithmId: u.EditValue.ManagerChainType.ToDasAlgorithmId(true),
			AddressHex:     u.EditValue.ManagerAddress,
			IsMulti:        false,
			ChainType:      u.EditValue.ManagerChainType,
		}
		args, err := daf.HexToArgs(ownerHex, managerHex)
		if err != nil {
			return fmt.Errorf("HexToArgs err: %s", err.Error())
		}
		record.EditArgs = common.Bytes2Hex(args)
	case common.EditKeyRecords:
		records := u.FormatRecords()
		record.EditRecords = toolib.JsonString(records)
	default:
		return fmt.Errorf("not exist edit key [%s]", u.EditKey)
	}
	return nil
}
