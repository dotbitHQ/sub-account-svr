package tables

import (
	"github.com/shopspring/decimal"
	"time"
)

type TableSubAccountAutoMintWithdrawHistory struct {
	Id                uint64          `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	TaskId            string          `json:"task_id" gorm:"column:task_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	ParentAccountId   string          `json:"parent_account_id" gorm:"column:parent_account_id; index:idx_parent_account_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	ServiceProviderId string          `json:"service_provider_id" gorm:"column:service_provider_id; index:idx_service_provider_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	TxHash            string          `json:"tx_hash" gorm:"column:tx_hash; index:idx_hash; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Price             decimal.Decimal `json:"price" gorm:"column:price; type:decimal(60,2) NOT NULL DEFAULT '0' COMMENT '';"`
	CreatedAt         time.Time       `json:"created_at" gorm:"column:created_at;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT ''"`
	UpdatedAt         time.Time       `json:"updated_at" gorm:"column:updated_at;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT ''"`
}

func (t *TableSubAccountAutoMintWithdrawHistory) TableName() string {
	return "t_sub_account_auto_mint_withdraw_history"
}
