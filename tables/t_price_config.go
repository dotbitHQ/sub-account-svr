package tables

import (
	"time"
)

type PriceConfig struct {
	Id        int64     `gorm:"column:id;type:bigint(20);primary_key;AUTO_INCREMENT" json:"id"`
	Account   string    `gorm:"column:account;type:varchar(255);comment:账号;NOT NULL" json:"account"`
	AccountId string    `gorm:"column:account_id;type:varchar(255);comment:账号id;NOT NULL" json:"account_id"`
	Action    string    `gorm:"column:action;type:varchar(20);comment:交易类型（price，blacklist, auto_mint）;NOT NULL" json:"action"`
	TxHash    string    `gorm:"column:tx_hash;type:varchar(255);comment:交易hash;NOT NULL" json:"tx_hash"`
	TxStatus  int       `gorm:"column:tx_status;type:smallint(6);default:0;comment:交易状态（0：未发起；1：进行中；2：完成；3：失败）;NOT NULL" json:"tx_status"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

func (m *PriceConfig) TableName() string {
	return "t_price_config"
}
