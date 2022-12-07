package dao

import "das_sub_account/tables"

func (d *DbDao) GetMinSignInfo(mintSignId string) (info tables.TableMintSignInfo, err error) {
	err = d.db.Where("mint_sign_id=?", mintSignId).Find(&info).Limit(1).Error
	return
}
