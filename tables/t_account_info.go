package tables

import (
	"github.com/dotbitHQ/das-lib/common"
	"time"
)

type TableAccountInfo struct {
	Id                   uint64                `json:"id" gorm:"column:id;primary_key;AUTO_INCREMENT"`
	BlockNumber          uint64                `json:"block_number" gorm:"column:block_number"`
	Outpoint             string                `json:"outpoint" gorm:"column:outpoint"`
	AccountId            string                `json:"account_id" gorm:"account_id"`
	Account              string                `json:"account" gorm:"column:account"`
	OwnerChainType       common.ChainType      `json:"owner_chain_type" gorm:"column:owner_chain_type"`
	Owner                string                `json:"owner" gorm:"column:owner"`
	OwnerAlgorithmId     common.DasAlgorithmId `json:"owner_algorithm_id" gorm:"column:owner_algorithm_id"`
	ManagerChainType     common.ChainType      `json:"manager_chain_type" gorm:"column:manager_chain_type"`
	Manager              string                `json:"manager" gorm:"column:manager"`
	ManagerAlgorithmId   common.DasAlgorithmId `json:"manager_algorithm_id" gorm:"column:manager_algorithm_id"`
	Status               AccountStatus         `json:"status" gorm:"column:status"`
	RegisteredAt         uint64                `json:"registered_at" gorm:"column:registered_at"`
	ExpiredAt            uint64                `json:"expired_at" gorm:"column:expired_at"`
	ConfirmProposalHash  string                `json:"confirm_proposal_hash" gorm:"column:confirm_proposal_hash"`
	ParentAccountId      string                `json:"parent_account_id" gorm:"column:parent_account_id"`
	EnableSubAccount     EnableSubAccount      `json:"enable_sub_account" gorm:"column:enable_sub_account"`
	RenewSubAccountPrice uint64                `json:"renew_sub_account_price" gorm:"column:renew_sub_account_price"`
	Nonce                uint64                `json:"nonce" gorm:"column:nonce"`
}

type AccountStatus int

const (
	AccountStatusNormal    AccountStatus = 0
	AccountStatusOnSale    AccountStatus = 1
	AccountStatusOnAuction AccountStatus = 2
	AccountStatusOnCross   AccountStatus = 3
	AccountStatusApproval  AccountStatus = 4
	TableNameAccountInfo                 = "t_account_info"
)

type EnableSubAccount = uint8

const (
	AccountEnableStatusOff EnableSubAccount = 0
	AccountEnableStatusOn  EnableSubAccount = 1
)

func (t *TableAccountInfo) TableName() string {
	return TableNameAccountInfo
}

func (t *TableAccountInfo) IsExpired() bool {
	if int64(t.ExpiredAt) <= time.Now().Unix() {
		return true
	}
	return false
}

// ============= account category ===============

type Category int

const (
	CategoryDefault          Category = 0
	CategoryMainAccount      Category = 1
	CategorySubAccount       Category = 2
	CategoryOnSale           Category = 3
	CategoryExpireSoon       Category = 4
	CategoryToBeRecycled     Category = 5
	CategoryEnableSubAccount Category = 6
)

type OrderType int

const (
	OrderTypeAccountAsc     OrderType = 0
	OrderTypeAccountDesc    OrderType = 1
	OrderTypeRegisterAtAsc  OrderType = 2
	OrderTypeRegisterAtDesc OrderType = 3
	OrderTypeExpiredAtAsc   OrderType = 4
	OrderTypeExpiredAtDesc  OrderType = 5
)
