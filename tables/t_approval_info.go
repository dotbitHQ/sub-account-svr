package tables

import (
	"time"
)

type ApprovalInfo struct {
	ID               uint64    `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	BlockNumber      uint64    `gorm:"column:block_number;default:0;NOT NULL"`
	Outpoint         string    `gorm:"column:outpoint;NOT NULL"` // Hash-Index
	Account          string    `gorm:"column:account;NOT NULL"`
	AccountID        string    `gorm:"column:account_id;NOT NULL"`
	ParentAccountID  string    `gorm:"column:parent_account_id;NOT NULL"`
	Status           int       `gorm:"column:status;default:0;NOT NULL"`             // 0-还未上链 1-开启授权 2:完成授权 3-撤销授权
	ProtectedUntil   uint64    `gorm:"column:protected_until;default:0;NOT NULL"`    // 授权的不可撤销时间
	SealedUntil      uint64    `gorm:"column:sealed_until;default:0;NOT NULL"`       // 授权开放时间
	DelayCountRemain int       `gorm:"column:delay_count_remain;default:0;NOT NULL"` // 可推迟次数
	PostponedTimes   int       `gorm:"column:postponed_times;default:0;NOT NULL"`    // 推迟过的次数
	CreatedAt        time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`
	UpdatedAt        time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`
}

func (m *ApprovalInfo) TableName() string {
	return "t_approval_info"
}
