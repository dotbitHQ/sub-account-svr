package tables

import (
	"crypto/md5"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"time"
)

type TableMintSignInfo struct {
	Id         uint64    `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	MintSignId string    `json:"mint_sign_id" gorm:"column:mint_sign_id; uniqueIndex:uk_mint_sign_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Signature  string    `json:"signature" gorm:"column:signature; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	SmtRoot    string    `json:"smt_root" gorm:"column:smt_root; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	ExpiredAt  uint64    `json:"expired_at" gorm:"column:expired_at; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	Timestamp  uint64    `json:"timestamp" gorm:"column:timestamp; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	KeyValue   string    `json:"key_value" gorm:"column:key_value; type:text NOT NULL COMMENT '';"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

const (
	TableNameMintSignInfo = "t_mint_sign_info"
)

func (t *TableMintSignInfo) TableName() string {
	return TableNameMintSignInfo
}

func (t *TableMintSignInfo) InitMintSignId(parentAccountId string) {
	mintSignId := fmt.Sprintf("%s%s%d%d", parentAccountId, t.SmtRoot, t.ExpiredAt, t.Timestamp)
	mintSignId = fmt.Sprintf("%x", md5.Sum([]byte(mintSignId)))
	t.MintSignId = mintSignId
}

func (t *TableMintSignInfo) GenWitness() []byte {
	sams := witness.SubAccountMintSign{
		Version:            witness.SubAccountMintSignVersion1,
		Signature:          common.Hex2Bytes(t.Signature),
		SignRole:           common.Hex2Bytes(common.ParamOwner),
		ExpiredAt:          t.ExpiredAt,
		AccountListSmtRoot: common.Hex2Bytes(t.SmtRoot),
	}
	return sams.GenSubAccountMintSignBytes()
}

// === KeyValue ===
type MintSignInfoKeyValue struct {
	Key   string `json:"k"`
	Value string `json:"v"`
}
