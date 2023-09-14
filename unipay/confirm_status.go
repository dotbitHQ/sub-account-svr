package unipay

import (
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"time"
)

func (t *ToolUniPay) RunConfirmStatus() {
	tickerSearchStatus := time.NewTicker(time.Minute * 5)

	t.Wg.Add(1)
	go func() {
		defer http_api.RecoverPanic()
		for {
			select {
			case <-tickerSearchStatus.C:
				log.Info("doConfirmStatus start")
				if err := t.doConfirmStatus(); err != nil {
					log.Errorf("doConfirmStatus err: %s", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doConfirmStatus", err.Error())
				}
				log.Info("doConfirmStatus end")
			case <-t.Ctx.Done():
				log.Info("RunRefund done")
				t.Wg.Done()
				return
			}
		}
	}()
}

func (t *ToolUniPay) doConfirmStatus() error {
	// for check order pay status
	pendingList, err := t.DbDao.GetPayHashStatusPendingList()
	if err != nil {
		return fmt.Errorf("GetPayHashStatusPendingList err: %s", err.Error())
	}
	var orderIdList []string
	for _, v := range pendingList {
		orderIdList = append(orderIdList, v.OrderId)
	}

	// for check refund status
	refundingList, err := t.DbDao.GetRefundStatusRefundingList()
	if err != nil {
		return fmt.Errorf("GetRefundStatusRefundingList err: %s", err.Error())
	}
	var payHashList []string
	for _, v := range refundingList {
		payHashList = append(payHashList, v.PayHash)
	}

	if len(orderIdList) == 0 && len(payHashList) == 0 {
		return nil
	}

	log.Info("doConfirmStatus:", len(orderIdList), len(payHashList))
	// call unipay
	resp, err := GetPaymentInfo(ReqPaymentInfo{
		BusinessId:  BusinessIdAutoSubAccount,
		OrderIdList: orderIdList,
		PayHashList: payHashList,
	})
	if err != nil {
		return fmt.Errorf("OrderInfo err: %s", err.Error())
	}

	var orderIdMap = make(map[string][]PaymentInfo)
	var payHashMap = make(map[string]PaymentInfo)
	for i, v := range resp.PaymentList {
		orderIdMap[v.OrderId] = append(orderIdMap[v.OrderId], resp.PaymentList[i])
		payHashMap[v.PayHash] = resp.PaymentList[i]
	}

	// payment confirm
	for _, pending := range pendingList {
		paymentInfoList, ok := orderIdMap[pending.OrderId]
		if !ok {
			min := pending.PayHashUnconfirmedMin()
			log.Info("PayHashUnconfirmedMin:", pending.OrderId, min)
			//if min > 10 {
			//	notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doConfirmStatus", pending.OrderId)
			//}
			//if min > 60 {
			//	if err := t.DbDao.UpdateOrderStatusToFailForUnconfirmedPayHash(pending.OrderId, pending.PayHash); err != nil {
			//		return fmt.Errorf("UpdateOrderStatusToFailForUnconfirmedPayHash err: %s", err.Error())
			//	}
			//}
			continue
		}
		for _, v := range paymentInfoList {
			if v.PayHashStatus != tables.PayHashStatusConfirmed {
				continue
			}
			if err = DoPaymentConfirm(t.DasCore, t.DbDao, v.OrderId, v.PayHash); err != nil {
				log.Errorf("DoPaymentConfirm err: %s", err.Error())
			}
		}
	}

	// refund confirm
	for _, v := range payHashList {
		paymentInfo, ok := payHashMap[v]
		if !ok {
			continue
		}
		if paymentInfo.RefundStatus != tables.RefundStatusRefunded {
			continue
		}
		if err = t.DbDao.UpdateRefundStatusToRefunded(paymentInfo.PayHash, paymentInfo.OrderId, paymentInfo.RefundHash); err != nil {
			log.Error("UpdateRefundStatusToRefunded err: ", err.Error())
		}
	}

	return nil
}

func DoPaymentConfirm(dasCore *core.DasCore, dbDao *dao.DbDao, orderId, payHash string) error {
	order, err := dbDao.GetOrderByOrderID(orderId)
	if err != nil {
		return fmt.Errorf("GetOrderByOrderID err: %s", err.Error())
	}
	if order.Id == 0 {
		return fmt.Errorf("order[%s] not exist", orderId)
	}

	smtRecord := tables.TableSmtRecordInfo{
		SvrName:         order.SvrName,
		AccountId:       order.AccountId,
		RecordType:      tables.RecordTypeDefault,
		MintType:        tables.MintTypeAutoMint,
		OrderID:         order.OrderId,
		Action:          common.DasActionUpdateSubAccount,
		ParentAccountId: tables.GetParentAccountId(order.Account),
		Account:         order.Account,
		Timestamp:       time.Now().UnixNano() / 1e6,
	}

	switch order.ActionType {
	case tables.ActionTypeMint:
		owner := core.DasAddressHex{
			DasAlgorithmId: order.AlgorithmId,
			AddressHex:     order.PayAddress,
		}
		args, err := dasCore.Daf().HexToArgs(owner, owner)
		if err != nil {
			return fmt.Errorf("HexToArgs err: %s", err.Error())
		}
		charsetList, err := dasCore.GetAccountCharSetList(order.Account)
		if err != nil {
			return fmt.Errorf("GetAccountCharSetList err: %s", err.Error())
		}
		content, err := json.Marshal(charsetList)
		if err != nil {
			return fmt.Errorf("json Marshal err: %s", err.Error())
		}
		smtRecord.RegisterYears = order.Years
		smtRecord.RegisterArgs = common.Bytes2Hex(args)
		smtRecord.Content = string(content)
		smtRecord.SubAction = common.SubActionCreate
	case tables.ActionTypeRenew:
		smtRecord.RenewYears = order.Years
		smtRecord.SubAction = common.SubActionRenew
		accInfo, err := dbDao.GetAccountInfoByAccountId(order.AccountId)
		if err != nil {
			return err
		}
		if accInfo.Id == 0 {
			return fmt.Errorf("account: [%s] no exist", order.Account)
		}
		smtRecord.Nonce = accInfo.Nonce + 1
	}

	paymentInfo := tables.PaymentInfo{
		PayHash:       payHash,
		OrderId:       orderId,
		PayHashStatus: tables.PayHashStatusConfirmed,
		Timestamp:     time.Now().UnixMilli(),
	}
	rowsAffected, err := dbDao.UpdateOrderPayStatusOkWithSmtRecord(paymentInfo, smtRecord)
	if err != nil {
		return fmt.Errorf("UpdateOrderPayStatusOkWithSmtRecord err: %s", err.Error())
	}
	if rowsAffected == 0 {
		log.Warnf("doUniPayNotice: %s %d", orderId, rowsAffected)
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "multiple orders success", orderId)
	}
	return nil
}
