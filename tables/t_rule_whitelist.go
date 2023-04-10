package tables

import "time"

type RuleType string

const (
	RuleTypePriceRules     RuleType = "price_rules"
	RuleTypePreservedRules RuleType = "preserved_rules"
)

type RuleWhitelist struct {
	Id              int64     `gorm:"column:id;AUTO_INCREMENT" json:"id"`
	TxHash          string    `gorm:"index:idx_tx_hash_acc_id;column:tx_hash;type:varchar(255);comment:交易hash;NOT NULL" json:"tx_hash"`
	ParentAccount   string    `gorm:"column:parent_account;type:varchar(255);comment:父账号;NOT NULL" json:"parent_account"`
	ParentAccountId string    `gorm:"index:idx_tx_hash_acc_id;column:parent_account_id;type:varchar(255);comment:父账号id;NOT NULL" json:"parent_account_id"`
	RuleType        RuleType  `gorm:"column:rule_type;type:varchar(255);comment:规则类型;NOT NULL" json:"rule_type"`
	RuleIndex       int       `gorm:"column:rule_index;type:int(11);comment:规则索引;NOT NULL" json:"rule_index"`
	Account         string    `gorm:"column:account;type:varchar(255);comment:账号;NOT NULL" json:"account"`
	AccountId       string    `gorm:"index:idx_tx_hash_acc_id;column:account_id;type:varchar(255);comment:账号id;NOT NULL" json:"account_id"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

func (m *RuleWhitelist) TableName() string {
	return "t_rule_white_list"
}
