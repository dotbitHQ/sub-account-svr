package tables

import (
	"crypto/md5"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"time"
)

type TableMintSignInfo struct {
	Id         uint64           `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	MintSignId string           `json:"mint_sign_id" gorm:"column:mint_sign_id; uniqueIndex:uk_mint_sign_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Signature  string           `json:"signature" gorm:"column:signature; type:text NOT NULL COMMENT '';"`
	SignRole   string           `json:"sign_role" gorm:"column:sign_role; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	SubAction  common.SubAction `json:"sub_action" gorm:"column:sub_action; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	SmtRoot    string           `json:"smt_root" gorm:"column:smt_root; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	ExpiredAt  uint64           `json:"expired_at" gorm:"column:expired_at; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	Timestamp  uint64           `json:"timestamp" gorm:"column:timestamp; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	KeyValue   string           `json:"key_value" gorm:"column:key_value; type:mediumtext NOT NULL COMMENT 'keyvalue';"`
	ChainType  common.ChainType `json:"chain_type" gorm:"column:chain_type; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '';"`
	Address    string           `json:"address" gorm:"column:address; index:k_address; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	CreatedAt  time.Time        `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt  time.Time        `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

const (
	TableNameMintSignInfo = "t_mint_sign_info"
)

func (t *TableMintSignInfo) TableName() string {
	return TableNameMintSignInfo
}

func (t *TableMintSignInfo) InitMintSignId(parentAccountId string) {
	mintSignId := fmt.Sprintf("%s%s%s%d%d", parentAccountId, t.SmtRoot, t.SubAction, t.ExpiredAt, t.Timestamp)
	mintSignId = fmt.Sprintf("%x", md5.Sum([]byte(mintSignId)))
	t.MintSignId = mintSignId
}

func (t *TableMintSignInfo) GenWitness() []byte {
	sams := witness.SubAccountMintSign{
		Version:            witness.SubAccountMintSignVersion1,
		Signature:          common.Hex2Bytes(t.Signature),
		SignRole:           common.Hex2Bytes(t.SignRole),
		ExpiredAt:          t.ExpiredAt,
		AccountListSmtRoot: common.Hex2Bytes(t.SmtRoot),
	}
	if t.SubAction == common.SubActionCreate {
		sams.SignRole = common.Hex2Bytes(common.ParamManager)
	}

	actionDataType := ""
	switch t.SubAction {
	case common.SubActionCreate:
		actionDataType = common.ActionDataTypeSubAccountMintSign
	case common.SubActionRenew:
		actionDataType = common.ActionDataTypeSubAccountRenewSign
	case common.SubActionCreateApproval:
		actionDataType = common.ActionDataTypeSubAccountCreateApprovalSign
	case common.SubActionDelayApproval:
		actionDataType = common.ActionDataTypeSubAccountDelayApprovalSign
	case common.SubActionRevokeApproval:
		actionDataType = common.ActionDataTypeSubAccountRevokeApprovalSign
	case common.SubActionFullfillApproval:
		actionDataType = common.ActionDataTypeSubAccountFulfillApprovalSign
	}
	return sams.GenWitnessWithAction(actionDataType)
}

// === KeyValue ===
type MintSignInfoKeyValue struct {
	Key   string `json:"k"`
	Value string `json:"v"`
}
