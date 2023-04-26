package tables

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

type PaymentStatus int

const (
	PaymentStatusDefault PaymentStatus = iota
	PaymentStatusPending
	PaymentStatusSuccess
	PaymentStatusFail
	PaymentStatusClosed
)

type AutoPaymentInfo struct {
	Id            int64           `gorm:"column:id;AUTO_INCREMENT" json:"id"`
	AutoPaymentId string          `gorm:"column:auto_payment_id;uniqueIndex:uk_auto_payment_id;type:varchar(255);comment:自动支付id;NOT NULL" json:"auto_payment_id"`
	Account       string          `gorm:"column:account;type:varchar(255);NOT NULL" json:"account"`
	AccountId     string          `gorm:"column:account_id;index:idx_account_id;type:varchar(255);NOT NULL" json:"account_id"`
	TokenId       string          `gorm:"column:token_id;type:varchar(255);comment:支付代币ID;NOT NULL" json:"token_id"`
	Amount        decimal.Decimal `gorm:"column:amount;type:decimal(60) unsigned;comment:付款金额;NOT NULL" json:"amount"`
	Address       string          `gorm:"column:address;type:varchar(255);comment:打款地址;NOT NULL" json:"address"`
	PaymentTx     string          `gorm:"column:payment_tx;type:varchar(255);comment:支付交易可能是交易hash或者PayPal等交易号;NOT NULL" json:"payment_tx"`
	PaymentDate   time.Time       `gorm:"column:payment_date;type:timestamp;comment:打款日期" json:"payment_date"`
	PaymentStatus PaymentStatus   `gorm:"column:payment_status;type:smallint(6);default:0;comment:0：默认；1：打款进处理中；2：交易成功；3：交易失败；4：交易关闭;NOT NULL" json:"payment_status"`
	CreatedAt     time.Time       `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

func (m *AutoPaymentInfo) TableName() string {
	return "t_auto_payment_info"
}

func (m *AutoPaymentInfo) GenAutoPaymentId() error {
	uid, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	uidStr := strings.ReplaceAll(uid.String(), "-", "")
	m.AutoPaymentId = uidStr
	return nil
}
