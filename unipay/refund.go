package unipay

import (
	"fmt"
)

func (t *ToolUniPay) doRefund() error {
	//get payment list
	list, err := t.DbDao.GetUnRefundList()
	if err != nil {
		return fmt.Errorf("GetUnRefundList err: %s", err.Error())
	}

	//call unipay to refund
	var req ReqOrderRefund
	req.BusinessId = BusinessIdAutoSubAccount
	var ids []uint64
	for _, v := range list {
		ids = append(ids, v.Id)
		req.RefundList = append(req.RefundList, RefundInfo{
			OrderId: v.OrderId,
			PayHash: v.PayHash,
		})
	}

	_, err = RefundOrder(req)
	if err != nil {
		return fmt.Errorf("RefundOrder err: %s", err.Error())
	}

	if err = t.DbDao.UpdateRefundStatusToRefundIng(ids); err != nil {
		return fmt.Errorf("UpdateRefundStatusToRefundIng err: %s", err.Error())
	}

	return nil
}
