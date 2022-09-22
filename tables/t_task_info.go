package tables

import (
	"crypto/md5"
	"fmt"
	"time"
)

type SmtStatus int

const (
	SmtStatusNeedToWrite      SmtStatus = 0
	SmtStatusWriting          SmtStatus = 1
	SmtStatusWriteComplete    SmtStatus = 2
	SmtStatusNeedToRollback   SmtStatus = 3
	SmtStatusRollbackComplete SmtStatus = 4
)

type TxStatus int

const (
	TxStatusUnSend    TxStatus = 0
	TxStatusPending   TxStatus = 1
	TxStatusCommitted TxStatus = 2
	TxStatusRejected  TxStatus = 3
)

type TaskType int

const (
	TaskTypeDelegate TaskType = 0 // if tx rejected, need retry
	TaskTypeNormal   TaskType = 1 // if tx rejected, closed
	TaskTypeChain    TaskType = 2
	TaskTypeClosed   TaskType = 3
)

type TableTaskInfo struct {
	Id              uint64    `json:"id" gorm:"column:id;primaryKey;type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT ''"`
	SvrName         string    `json:"svr_name" gorm:"column:svr_name; index:k_svr_name; type:varchar(255) NOT NULL DEFAULT '' COMMENT 'smt tree';"`
	TaskId          string    `json:"task_id" gorm:"column:task_id;uniqueIndex:uk_task_id;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	TaskType        TaskType  `json:"task_type" gorm:"column:task_type;index:k_task_type;type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-delegate 1-normal 2-chain 3-closed'"`
	ParentAccountId string    `json:"parent_account_id" gorm:"column:parent_account_id;index:k_parent_account_id;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT 'smt tree'"`
	Action          string    `json:"action" gorm:"column:action;index:k_action;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	RefOutpoint     string    `json:"ref_outpoint" gorm:"column:ref_outpoint;index:k_ref_outpoint;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT 'ref sub account cell outpoint'"`
	BlockNumber     uint64    `json:"block_number" gorm:"column:block_number;type:bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'tx block number'"`
	Outpoint        string    `json:"outpoint" gorm:"column:outpoint;index:k_outpoint;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT 'new sub account cell outpoint'"`
	Timestamp       int64     `json:"timestamp" gorm:"column:timestamp;type:bigint(20) NOT NULL DEFAULT '0' COMMENT 'record timestamp'"`
	SmtStatus       SmtStatus `json:"smt_status" gorm:"column:smt_status;index:k_smt_tx;type:smallint(6) NOT NULL DEFAULT '0' COMMENT 'smt status'"`
	TxStatus        TxStatus  `json:"tx_status" gorm:"column:tx_status;index:k_smt_tx;type:smallint(6) NOT NULL DEFAULT '0' COMMENT 'tx status'"`
	Retry           int       `json:"retry" gorm:"column:retry;type:smallint(6) NOT NULL DEFAULT '0' COMMENT ''"`
	CustomScripHash string    `json:"custom_scrip_hash" gorm:"column:custom_scrip_hash; type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT ''"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT ''"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT ''"`
}

const (
	TableNameTaskInfo = "t_task_info"
)

func (t *TableTaskInfo) TableName() string {
	return TableNameTaskInfo
}

func (t *TableTaskInfo) InitTaskId() {
	taskId := fmt.Sprintf("%s%s%s%d", t.ParentAccountId, t.Action, t.Outpoint, t.Timestamp)
	taskId = fmt.Sprintf("%x", md5.Sum([]byte(taskId)))
	t.TaskId = taskId
	time.Sleep(time.Millisecond * 100)
}
