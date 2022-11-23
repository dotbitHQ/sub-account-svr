package tables

import "time"

type TableMintSignInfo struct {
	Id         uint64    `json:"id" gorm:"column:id;primaryKey;type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT ''"`
	MintSignId string    `json:"mint_sign_id" gorm:"column:mint_sign_id; uniqueIndex:uk_mint_sign_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Signature  string    `json:"signature" gorm:"column:signature; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

const (
	TableNameMintSignInfo = "t_mint_sign_info"
)

func (t *TableMintSignInfo) TableName() string {
	return TableNameMintSignInfo
}
