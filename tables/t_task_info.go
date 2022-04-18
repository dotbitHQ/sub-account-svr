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
	Id              uint64    `json:"id" gorm:"column:id;primary_key;AUTO_INCREMENT"`
	TaskId          string    `json:"task_id" gorm:"column:task_id"`
	TaskType        TaskType  `json:"task_type" gorm:"column:task_type"`
	ParentAccountId string    `json:"parent_account_id" gorm:"column:parent_account_id"`
	Action          string    `json:"action" gorm:"column:action"`
	RefOutpoint     string    `json:"ref_outpoint" gorm:"column:ref_outpoint"`
	BlockNumber     uint64    `json:"block_number" gorm:"column:block_number"`
	Outpoint        string    `json:"outpoint" gorm:"column:outpoint"`
	Timestamp       int64     `json:"timestamp" gorm:"column:timestamp"`
	SmtStatus       SmtStatus `json:"smt_status" gorm:"column:smt_status"`
	TxStatus        TxStatus  `json:"tx_status" gorm:"column:tx_status"`
	Retry           int       `json:"retry" gorm:"column:retry"`
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
