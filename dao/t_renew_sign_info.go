package dao

import "das_sub_account/tables"

func (d *DbDao) GetRenewSignInfo(renewSignId string) (info tables.TableRenewSignInfo, err error) {
	err = d.db.Where("renew_sign_id=?", renewSignId).First(&info).Error
	return
}
