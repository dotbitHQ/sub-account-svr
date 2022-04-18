package tables

type TableSmtInfo struct {
	Id              uint64 `json:"id" gorm:"column:id"`
	BlockNumber     uint64 `json:"block_number" gorm:"column:block_number"`
	Outpoint        string `json:"outpoint" gorm:"column:outpoint"`
	AccountId       string `json:"account_id" gorm:"column:account_id"`
	ParentAccountId string `json:"parent_account_id" gorm:"column:parent_account_id"`
	LeafDataHash    string `json:"leaf_data_hash" gorm:"column:leaf_data_hash"`
}

const (
	TableNameSmtInfo = "t_smt_info"
)

func (t *TableSmtInfo) TableName() string {
	return TableNameSmtInfo
}
