package tables

import (
	"crypto/md5"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"time"
)

type TableRenewSignInfo struct {
	Id          uint64           `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	RenewSignId string           `json:"renew_sign_id" gorm:"column:renew_sign_id; uniqueIndex:uk_renew_sign_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Signature   string           `json:"signature" gorm:"column:signature; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	SignRole    string           `json:"sign_role" gorm:"column:sign_role; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	SmtRoot     string           `json:"smt_root" gorm:"column:smt_root; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	ExpiredAt   uint64           `json:"expired_at" gorm:"column:expired_at; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	Timestamp   uint64           `json:"timestamp" gorm:"column:timestamp; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	KeyValue    string           `json:"key_value" gorm:"column:key_value; type:mediumtext NOT NULL COMMENT 'keyvalue';"`
	ChainType   common.ChainType `json:"chain_type" gorm:"column:chain_type; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '';"`
	Address     string           `json:"address" gorm:"column:address; index:k_address; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	CreatedAt   time.Time        `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt   time.Time        `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

const (
	TableNameRenewSignInfo = "t_renew_sign_info"
)

func (t *TableRenewSignInfo) TableName() string {
	return TableNameRenewSignInfo
}

func (t *TableRenewSignInfo) InitMintSignId(parentAccountId string) {
	mintSignId := fmt.Sprintf("%s%s%d%d%s", parentAccountId, t.SmtRoot, t.ExpiredAt, t.Timestamp, t.Address)
	mintSignId = fmt.Sprintf("%x", md5.Sum([]byte(mintSignId)))
	t.RenewSignId = mintSignId
}

func (t *TableRenewSignInfo) GenWitness() []byte {
	sams := witness.SubAccountRenewSign{
		Version:            witness.SubAccountMintSignVersion1,
		Signature:          common.Hex2Bytes(t.Signature),
		SignRole:           common.Hex2Bytes(t.SignRole),
		ExpiredAt:          t.ExpiredAt,
		AccountListSmtRoot: common.Hex2Bytes(t.SmtRoot),
	}
	return sams.GenWitness()
}

// RenewSignInfoKeyValue === KeyValue ===
type RenewSignInfoKeyValue struct {
	Key   string `json:"k"`
	Value string `json:"v"`
}
