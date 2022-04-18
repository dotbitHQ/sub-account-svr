package tables

import (
	"encoding/json"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
	"github.com/DeAccountSystems/das-lib/witness"
	"github.com/scorpiotzh/mylog"
	"strings"
)

var log = mylog.NewLogger("tables", mylog.LevelDebug)

type TableSmtRecordInfo struct {
	Id              uint64     `json:"id" gorm:"column:id;primary_key;AUTO_INCREMENT"`
	AccountId       string     `json:"account_id" gorm:"column:account_id"`
	Nonce           uint64     `json:"nonce" gorm:"column:nonce"`
	RecordType      RecordType `json:"record_type" gorm:"column:record_type"`
	TaskId          string     `json:"task_id" gorm:"column:task_id"`
	Action          string     `json:"action" gorm:"column:action"`
	ParentAccountId string     `json:"parent_account_id" gorm:"column:parent_account_id"`
	Account         string     `json:"account" gorm:"column:account"`
	RegisterYears   uint64     `json:"register_years" gorm:"column:register_years"`
	RegisterArgs    string     `json:"register_args" gorm:"column:register_args"`
	EditKey         string     `json:"edit_key" gorm:"column:edit_key"`
	Signature       string     `json:"signature" gorm:"column:signature"`
	EditArgs        string     `json:"edit_args" gorm:"column:edit_args"`
	RenewYears      uint64     `json:"renew_years" gorm:"column:renew_years"`
	EditRecords     string     `json:"edit_records" gorm:"column:edit_records"`
	Timestamp       int64      `json:"timestamp" gorm:"column:timestamp"`
}

const (
	TableNameSmtRecordInfo = "t_smt_record_info"
)

func (t *TableSmtRecordInfo) TableName() string {
	return TableNameSmtRecordInfo
}

type RecordType int

const (
	RecordTypeDefault RecordType = 0
	RecordTypeClosed  RecordType = 1
	RecordTypeChain   RecordType = 2
)

func (t *TableSmtRecordInfo) getEditRecords() (records []witness.SubAccountRecord, err error) {
	err = json.Unmarshal([]byte(t.EditRecords), &records)
	return
}

func (t *TableSmtRecordInfo) GetCurrentSubAccount(oldSubAccount *witness.SubAccount, contractDas *core.DasContractInfo, timeCellTimestamp int64) (*witness.SubAccount, *witness.SubAccountParam, error) {
	var currentSubAccount witness.SubAccount
	var subAccountParam witness.SubAccountParam

	if contractDas == nil {
		return nil, nil, fmt.Errorf("contractDas is nil")
	}

	switch t.Action {
	case common.DasActionCreateSubAccount:
		accountCharSet, err := common.AccountToAccountChars(t.Account[:strings.Index(t.Account, ".")])
		if err != nil {
			return nil, nil, fmt.Errorf("AccountToAccountChars err: %s", err.Error())
		}
		currentSubAccount.Lock = contractDas.ToScript(common.Hex2Bytes(t.RegisterArgs))
		currentSubAccount.AccountId = t.AccountId
		currentSubAccount.AccountCharSet = accountCharSet
		currentSubAccount.Suffix = t.Account[strings.Index(t.Account, "."):]
		currentSubAccount.RegisteredAt = uint64(timeCellTimestamp)
		currentSubAccount.ExpiredAt = currentSubAccount.RegisteredAt + (31536000 * t.RegisterYears)

		subAccountParam.SubAccount = &currentSubAccount
		return &currentSubAccount, &subAccountParam, nil
	case common.DasActionEditSubAccount, common.DasActionRenewSubAccount:
		if oldSubAccount == nil {
			return nil, nil, fmt.Errorf("oldSubAccount is nil")
		}
		currentSubAccount = *oldSubAccount

		subAccountParam.Signature = common.Hex2Bytes(t.Signature)
		subAccountParam.SubAccount = oldSubAccount
		subAccountParam.EditKey = t.EditKey
		switch t.EditKey {
		case common.EditKeyOwner:
			currentSubAccount.Lock = contractDas.ToScript(common.Hex2Bytes(t.EditArgs))
			subAccountParam.SignRole = common.Hex2Bytes(common.ParamOwner)
			subAccountParam.EditLockArgs = common.Hex2Bytes(t.EditArgs)
			currentSubAccount.Records = nil
		case common.EditKeyManager:
			currentSubAccount.Lock = contractDas.ToScript(common.Hex2Bytes(t.EditArgs))
			subAccountParam.SignRole = common.Hex2Bytes(common.ParamOwner)
			subAccountParam.EditLockArgs = common.Hex2Bytes(t.EditArgs)
		case common.EditKeyRecords:
			records, err := t.getEditRecords()
			if err != nil {
				return nil, nil, fmt.Errorf("getEditRecords err: %s", err.Error())
			}
			currentSubAccount.Records = records
			subAccountParam.SignRole = common.Hex2Bytes(common.ParamManager)
			subAccountParam.EditRecords = records
		case common.EditKeyExpiredAt:
			currentSubAccount.ExpiredAt += 31536000 * t.RegisterYears
			subAccountParam.RenewExpiredAt = currentSubAccount.ExpiredAt
		default:
			return nil, nil, fmt.Errorf("not supported edit key[%s]", t.Action)
		}
		currentSubAccount.Nonce++
		return &currentSubAccount, &subAccountParam, nil
	default:
		return nil, nil, fmt.Errorf("not supported action[%s]", t.Action)
	}
}
