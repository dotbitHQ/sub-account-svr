package tables

import (
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/scorpiotzh/mylog"
	"strings"
	"time"
)

var log = mylog.NewLogger("tables", mylog.LevelDebug)

type TableSmtRecordInfo struct {
	Id              uint64           `json:"id" gorm:"column:id;primaryKey;type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT ''"`
	SvrName         string           `json:"svr_name" gorm:"column:svr_name; index:k_svr_name; type:varchar(255) NOT NULL DEFAULT '' COMMENT 'smt tree';"`
	AccountId       string           `json:"account_id" gorm:"column:account_id;uniqueIndex:uk_acc_nonce_bn;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	Nonce           uint64           `json:"nonce" gorm:"column:nonce;uniqueIndex:uk_acc_nonce_bn;type:int(11) NOT NULL DEFAULT '0' COMMENT ''"`
	RecordType      RecordType       `json:"record_type" gorm:"column:record_type;uniqueIndex:uk_acc_nonce_bn;type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-normal 1-closed 2-chain'"`
	RecordBN        uint64           `json:"record_bn" gorm:"column:record_bn; uniqueIndex:uk_acc_nonce_bn; type:bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT ''; "`
	MintType        MintType         `json:"mint_type" gorm:"column:mint_type;type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-default, 1-manually mint, 2-script mint, 3-auto mint';"`
	OrderID         string           `json:"order_id" gorm:"column:order_id;index:idx_order_id;type:varchar(255) NOT NULL DEFAULT '' COMMENT 'auto mint orderId';"`
	TaskId          string           `json:"task_id" gorm:"column:task_id;index:k_task_id;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	Action          string           `json:"action" gorm:"column:action;index:k_action;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	ParentAccountId string           `json:"parent_account_id" gorm:"column:parent_account_id;index:k_parent_account_id;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT 'smt tree'"`
	Account         string           `json:"account" gorm:"column:account;index:k_account;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	Content         string           `json:"content" gorm:"column:content;type:text NOT NULL COMMENT 'account char set'"`
	RegisterYears   uint64           `json:"register_years" gorm:"column:register_years;type:int(11) NOT NULL DEFAULT '0' COMMENT ''"`
	RegisterArgs    string           `json:"register_args" gorm:"column:register_args;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	EditKey         string           `json:"edit_key" gorm:"column:edit_key;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT 'owner,manager,records'"`
	EditValue       string           `json:"edit_value" gorm:"type:text;column:edit_value"`
	SignRole        string           `json:"sign_role" gorm:"column:sign_role;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT 'owner,manager,records'"`
	Signature       string           `json:"signature" gorm:"column:signature;type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	LoginChainType  common.ChainType `json:"login_chain_type" gorm:"column:login_chain_type"`
	LoginAddress    string           `json:"login_address" gorm:"column:login_address; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	SignAddress     string           `json:"sign_address" gorm:"column:sign_address; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	EditArgs        string           `json:"edit_args" gorm:"column:edit_args;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	RenewYears      uint64           `json:"renew_years" gorm:"column:renew_years;type:int(11) NOT NULL DEFAULT '0' COMMENT ''"`
	EditRecords     string           `json:"edit_records" gorm:"column:edit_records;type:text NOT NULL COMMENT ''"`
	Timestamp       int64            `json:"timestamp" gorm:"column:timestamp;type:bigint(20) NOT NULL DEFAULT '0' COMMENT 'record timestamp'"`
	SubAction       common.SubAction `json:"sub_action" gorm:"column:sub_action; index:k_action; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	MintSignId      string           `json:"mint_sign_id" gorm:"column:mint_sign_id; index:k_mint_sign_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	ExpiredAt       uint64           `json:"expired_at" gorm:"column:expired_at; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	CreatedAt       time.Time        `json:"created_at" gorm:"column:created_at;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT ''"`
	UpdatedAt       time.Time        `json:"updated_at" gorm:"column:updated_at;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT ''"`
}

const (
	TableNameSmtRecordInfo = "t_smt_record_info"
)

func (t *TableSmtRecordInfo) TableName() string {
	return TableNameSmtRecordInfo
}

type RecordType int

type MintType uint32

const (
	RecordTypeDefault RecordType = 0
	RecordTypeClosed  RecordType = 1
	RecordTypeChain   RecordType = 2

	MintTypeDefault      MintType = 0
	MintTypeManual       MintType = 1
	MintTypeCustomScript MintType = 2
	MintTypeAutoMint     MintType = 3
)

func (t *TableSmtRecordInfo) getEditRecords() (records []witness.Record, err error) {
	err = json.Unmarshal([]byte(t.EditRecords), &records)
	return
}
func (t *TableSmtRecordInfo) GetCurrentSubAccountNew(dasCore *core.DasCore, oldSubAccount *witness.SubAccountNew, contractDas *core.DasContractInfo, timeCellTimestamp int64) (*witness.SubAccountData, *witness.SubAccountNew, error) {
	if contractDas == nil {
		return nil, nil, fmt.Errorf("contractDas is nil")
	}

	var currentSubAccount witness.SubAccountData
	var subAccountNew witness.SubAccountNew
	subAccountNew.Action = t.SubAction
	subAccountNew.Version = witness.SubAccountNewVersion3
	subAccountNew.OldSubAccountVersion = witness.SubAccountVersionLatest
	subAccountNew.NewSubAccountVersion = witness.SubAccountVersionLatest

	if oldSubAccount != nil {
		if oldSubAccount.OldSubAccountVersion == 0 &&
			oldSubAccount.NewSubAccountVersion == 0 {
			subAccountNew.OldSubAccountVersion = witness.SubAccountVersion1
			subAccountNew.NewSubAccountVersion = witness.SubAccountVersion2
		}
		if oldSubAccount.OldSubAccountVersion == witness.SubAccountVersion1 &&
			oldSubAccount.NewSubAccountVersion == witness.SubAccountVersion2 {
			subAccountNew.OldSubAccountVersion = witness.SubAccountVersion2
			subAccountNew.NewSubAccountVersion = witness.SubAccountVersion2
		}
	}

	switch t.Action {
	case common.DasActionUpdateSubAccount:
		switch t.SubAction {
		case common.SubActionCreate:
			var accountCharSet []common.AccountCharSet
			if err := json.Unmarshal([]byte(t.Content), &accountCharSet); err != nil {
				return nil, nil, fmt.Errorf("json Unmarshal err: %s", err.Error())
			}
			currentSubAccount.Lock = contractDas.ToScript(common.Hex2Bytes(t.RegisterArgs))
			currentSubAccount.AccountId = t.AccountId
			currentSubAccount.AccountCharSet = accountCharSet
			currentSubAccount.Suffix = t.Account[strings.Index(t.Account, "."):]
			currentSubAccount.RegisteredAt = uint64(timeCellTimestamp)
			currentSubAccount.ExpiredAt = currentSubAccount.RegisteredAt + (uint64(common.OneYearSec) * t.RegisterYears)
			// default records
			oHex, _, err := dasCore.Daf().ScriptToHex(currentSubAccount.Lock)
			if err != nil {
				return nil, nil, fmt.Errorf("default records ScriptToHex err: %s", err.Error())
			}
			if coinType := oHex.DasAlgorithmId.ToCoinType(); coinType != "" {
				addrNormal, err := dasCore.Daf().HexToNormal(oHex)
				if err != nil {
					return nil, nil, fmt.Errorf("default records HexToNormal err: %s", err.Error())
				}
				currentSubAccount.Records = append(currentSubAccount.Records, witness.Record{
					Key:   string(coinType),
					Type:  "address",
					Label: "",
					Value: addrNormal.AddressNormal,
					TTL:   300,
				})
			}
			subAccountNew.SubAccountData = &currentSubAccount
			return &currentSubAccount, &subAccountNew, nil
		case common.SubActionRenew:
			if oldSubAccount == nil {
				return nil, nil, fmt.Errorf("oldSubAccount is nil")
			}
			currentSubAccount = *oldSubAccount.CurrentSubAccountData
			currentSubAccount.Nonce++
			currentSubAccount.ExpiredAt += uint64(common.OneYearSec) * t.RenewYears

			subAccountNew.EditKey = t.EditKey
			subAccountNew.SubAccountData = oldSubAccount.CurrentSubAccountData
			expiredAt := molecule.GoU64ToMoleculeU64(currentSubAccount.ExpiredAt)
			subAccountNew.EditValue = expiredAt.AsSlice()
			return &currentSubAccount, &subAccountNew, nil
		case common.SubActionEdit:
			if oldSubAccount == nil {
				return nil, nil, fmt.Errorf("oldSubAccount is nil")
			}
			currentSubAccount = *oldSubAccount.CurrentSubAccountData

			subAccountNew.Signature = common.Hex2Bytes(t.Signature)
			subAccountNew.SubAccountData = oldSubAccount.CurrentSubAccountData
			subAccountNew.EditKey = t.EditKey
			switch t.EditKey {
			case common.EditKeyOwner:
				currentSubAccount.Lock = contractDas.ToScript(common.Hex2Bytes(t.EditArgs))
				subAccountNew.SignRole = common.Hex2Bytes(common.ParamOwner)
				subAccountNew.EditLockArgs = common.Hex2Bytes(t.EditArgs)
				currentSubAccount.Records = nil
			case common.EditKeyManager:
				currentSubAccount.Lock = contractDas.ToScript(common.Hex2Bytes(t.EditArgs))
				subAccountNew.SignRole = common.Hex2Bytes(common.ParamOwner)
				subAccountNew.EditLockArgs = common.Hex2Bytes(t.EditArgs)
			case common.EditKeyRecords:
				records, err := t.getEditRecords()
				if err != nil {
					return nil, nil, fmt.Errorf("getEditRecords err: %s", err.Error())
				}
				currentSubAccount.Records = records
				subAccountNew.SignRole = common.Hex2Bytes(common.ParamManager)
				subAccountNew.EditRecords = records
			//case common.EditKeyExpiredAt:
			//	currentSubAccount.ExpiredAt += 31536000 * t.RegisterYears
			//	subAccountNew.RenewExpiredAt = currentSubAccount.ExpiredAt
			default:
				return nil, nil, fmt.Errorf("not supported edit key[%s]", t.Action)
			}
			currentSubAccount.Nonce++
			subAccountNew.SignExpiredAt = t.ExpiredAt
			return &currentSubAccount, &subAccountNew, nil
		case common.SubActionRecycle:
			if oldSubAccount == nil {
				return nil, nil, fmt.Errorf("oldSubAccount is nil")
			}
			subAccountNew.SubAccountData = oldSubAccount.CurrentSubAccountData
			return &currentSubAccount, &subAccountNew, nil
		case common.SubActionCreateApproval, common.SubActionDelayApproval,
			common.SubActionRevokeApproval, common.SubActionFullfillApproval:
			if oldSubAccount == nil {
				return nil, nil, fmt.Errorf("oldSubAccount is nil")
			}
			currentSubAccount = *oldSubAccount.CurrentSubAccountData

			switch t.SubAction {
			case common.SubActionCreateApproval:
				currentSubAccount.Status = common.AccountStatusOnApproval
			case common.SubActionRevokeApproval:
				currentSubAccount.Status = common.AccountStatusNormal
				currentSubAccount.AccountApproval = witness.AccountApproval{}
			case common.SubActionFullfillApproval:
				currentSubAccount.Status = common.AccountStatusNormal
				currentSubAccount.Lock = oldSubAccount.CurrentSubAccountData.AccountApproval.Params.Transfer.ToLock
				currentSubAccount.AccountApproval = witness.AccountApproval{}
				currentSubAccount.Records = []witness.Record{}
			}
			currentSubAccount.Nonce++
			if t.SignRole != "" {
				subAccountNew.SignRole = common.Hex2Bytes(t.SignRole)
			}
			subAccountNew.Signature = common.Hex2Bytes(t.Signature)
			subAccountNew.SignExpiredAt = t.ExpiredAt
			subAccountNew.SubAccountData = oldSubAccount.CurrentSubAccountData
			subAccountNew.EditKey = t.EditKey
			if t.EditValue != "" {
				subAccountNew.EditValue = common.Hex2Bytes(t.EditValue)
				accApproval, err := witness.AccountApprovalFromSlice(subAccountNew.EditValue)
				if err != nil {
					return nil, nil, err
				}
				currentSubAccount.AccountApproval = *accApproval
			}
			return &currentSubAccount, &subAccountNew, nil
		default:
			return nil, nil, fmt.Errorf("unknow sub-action[%s]", t.SubAction)
		}
	default:
		return nil, nil, fmt.Errorf("unknow action[%s]", t.Action)
	}
}
