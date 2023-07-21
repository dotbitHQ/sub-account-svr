package tables

import "time"

type ApprovalInfo struct {
	ID               uint64    `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	BlockNumber      uint64    `gorm:"column:block_number;default:0;NOT NULL"`
	Outpoint         string    `gorm:"column:outpoint;NOT NULL"` // Hash-Index
	Account          string    `gorm:"column:account;NOT NULL"`
	AccountID        string    `gorm:"column:account_id;NOT NULL"`
	ParentAccountID  string    `gorm:"column:parent_account_id;NOT NULL"`
	Action           string    `gorm:"column:action;NOT NULL"`
	Platform         string    `gorm:"column:platform;NOT NULL"` // platform address
	OwnerAlgorithmID int       `gorm:"column:owner_algorithm_id;default:0;NOT NULL"`
	Owner            string    `gorm:"column:owner;NOT NULL"` // owner address
	ToAlgorithmID    int       `gorm:"column:to_algorithm_id;default:0;NOT NULL"`
	To               string    `gorm:"column:to;NOT NULL"`                        // to address
	ProtectedUntil   uint64    `gorm:"column:protected_until;default:0;NOT NULL"` // 授权的不可撤销时间
	SealedUntil      uint64    `gorm:"column:sealed_until;default:0;NOT NULL"`    // 授权开放时间
	MaxDelayCount    int       `gorm:"column:max_delay_count;default:0;NOT NULL"` // 可推迟次数
	PostponedCount   int       `gorm:"column:postponed_count;default:0;NOT NULL"` // 推迟过的次数
	Status           int       `gorm:"column:status;default:0;NOT NULL"`          // 0-default 1-开启授权 2:完成授权 3-撤销授权
	CreatedAt        time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

func (m *ApprovalInfo) TableName() string {
	return "t_approval_info"
}
