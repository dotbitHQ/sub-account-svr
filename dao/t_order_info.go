package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

type OrderAndPaymentInfo struct {
	Id        int64   `gorm:"column:id" json:"id"`
	Account   string  `gorm:"column:account" json:"account"`
	AccountId string  `gorm:"column:account_id" json:"account_id"`
	TokenId   string  `gorm:"column:token_id" json:"token_id"`
	Amount    float64 `gorm:"column:amount" json:"amount"`
}

func (d *DbDao) GetOrderByOrderID(orderID string) (order tables.OrderInfo, err error) {
	err = d.db.Where("order_id=?", orderID).First(&order).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) FindOrderPaymentInfo(begin, end string, account ...string) (list []OrderAndPaymentInfo, err error) {
	db := d.db.Raw(`
select t1.id as id,t1.parent_account as account,t1.parent_account_id as account_id,t1.token_id as token_id,t1.amount as amount from (
SELECT t1.*,t2.token_id,t2.address as payment_address,t2.amount FROM
		t_order_info as t1
		LEFT JOIN (
		SELECT
			t1.*
		FROM
			t_payment_info t1
			INNER JOIN ( SELECT order_id, MIN( id ) AS min_id FROM t_payment_info GROUP BY order_id ) t2 ON t1.order_id = t2.order_id
			AND t1.id = t2.min_id where t1.pay_status=? and t1.refund_status=0 and t1.cancel_status=0
		 ) AS t2 ON t1.order_id = t2.order_id
	WHERE
		t1.order_status = ?
		AND t1.created_at >= ?
		AND t1.created_at <= ?) as t1`, tables.PayStatusSuccess, tables.OrderStatusSuccess, begin, end)
	if len(account) > 0 && account[0] != "" {
		db = db.Where("account=?", account[0])
	}
	err = db.Scan(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) UpdateAutoPaymentIdById(ids []int64, paymentId string) error {
	if len(ids) <= 0 {
		return nil
	}
	return d.db.Model(&tables.OrderInfo{}).Where("id in (?)", ids).Updates(map[string]interface{}{
		"auto_payment_id": paymentId,
	}).Error
}
