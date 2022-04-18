package tables

type TableBlockParserInfo struct {
	Id          uint64     `json:"id" gorm:"column:id;primary_key;AUTO_INCREMENT"`
	ParserType  ParserType `json:"parser_type" gorm:"column:parser_type"`
	BlockNumber uint64     `json:"block_number" gorm:"column:block_number"`
	BlockHash   string     `json:"block_hash" gorm:"column:block_hash"`
	ParentHash  string     `json:"parent_hash" gorm:"column:parent_hash"`
}

const (
	TableNameBlockParserInfo = "t_block_parser_info"
)

func (t *TableBlockParserInfo) TableName() string {
	return TableNameBlockParserInfo
}

type ParserType int

const (
	ParserTypeDASRegister = 99 // das-register
	ParserTypeSubAccount  = 98 // das-sub-account
	ParserTypeCKB         = 0
	ParserTypeETH         = 1
	ParserTypeTRON        = 3
	ParserTypeBSC         = 5
	ParserTypePOLYGON     = 6
)
