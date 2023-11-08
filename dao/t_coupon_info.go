package dao

import (
	"das_sub_account/config"
	"das_sub_account/encrypt"
	"das_sub_account/tables"
	"errors"
	"gorm.io/gorm"
)

func (d *DbDao) CouponExists(codes map[string]struct{}) ([]string, error) {
	codeAry := make([]string, 0)
	for v := range codes {
		codeAry = append(codeAry, v)
	}

	find := make([]*tables.CouponInfo, 0)
	if err := d.db.Where("code in (?)", codeAry).Find(&find).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []string{}, nil
		}
		return nil, err
	}

	res := make([]string, 0)
	for _, v := range find {
		res = append(res, v.Code)
	}
	return res, nil
}

func (d *DbDao) CreateCoupon(set *tables.CouponSetInfo, codes []tables.CouponInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(set).Error; err != nil {
			return err
		}
		for idx := range codes {
			if err := tx.Create(&codes[idx]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DbDao) GetCouponSetInfo(cid string) (res tables.CouponSetInfo, err error) {
	if err = d.db.Where("cid = ?", cid).First(&res).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
			return
		}
	}
	return
}

func (d *DbDao) GetCouponSetInfoByOrderId(orderId string) (res tables.CouponSetInfo, err error) {
	if err = d.db.Where("order_id = ?", orderId).First(&res).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
			return
		}
	}
	return
}

func (d *DbDao) FindCouponByCid(cid string) ([]*tables.CouponInfo, error) {
	res := make([]*tables.CouponInfo, 0)
	if err := d.db.Where("cid = ?", cid).Find(&res).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, nil
		}
		return nil, err
	}
	return res, nil
}

func (d *DbDao) UpdateCouponSetInfo(setInfo *tables.CouponSetInfo) error {
	return d.db.Save(setInfo).Error
}

func (d *DbDao) GetUnPaidCouponSetByAccId(accId string) (res tables.CouponSetInfo, err error) {
	if err = d.db.Where("account_id = ? and status = ?", accId, tables.CouponSetInfoStatusPending).First(&res).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
			return
		}
	}
	return
}

func (d *DbDao) FindCouponSetInfoList(accId string, page, pageSize int) ([]*tables.CouponSetInfo, int64, error) {
	var total int64
	res := make([]*tables.CouponSetInfo, 0)

	db := d.db.Model(&tables.CouponSetInfo{}).Where("account_id = ?", accId)
	if err := db.Count(&total).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, err
		}
		return res, 0, nil
	}

	if err := db.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&res).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, err
		}
	}
	return res, total, nil
}

func (d *DbDao) FindCouponCodeList(cid string, page, pageSize int) (res []*tables.CouponInfo, total int64, used int64, err error) {
	db := d.db.Model(&tables.CouponInfo{}).Where("cid = ?", cid)
	if err = db.Count(&total).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
		}
		return
	}
	if err = d.db.Model(&tables.CouponInfo{}).Where("cid=? and status=?", cid, tables.CouponStatusUsed).Count(&used).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		err = nil
	}
	if err = db.Order("status asc").Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&res).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
		}
		return
	}
	for idx, v := range res {
		res[idx].Code, err = encrypt.AesDecrypt(v.Code, config.Cfg.Das.Coupon.EncryptionKey)
		if err != nil {
			return
		}
	}
	return
}

func (d *DbDao) FindCouponCode(cid string) (res []*tables.CouponInfo, err error) {
	if err = d.db.Where("cid = ?", cid).Order("created_at desc").Find(&res).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
		}
	}
	for idx, v := range res {
		res[idx].Code, err = encrypt.AesDecrypt(v.Code, config.Cfg.Das.Coupon.EncryptionKey)
		if err != nil {
			return
		}
	}
	return res, nil
}

func (d *DbDao) GetCouponByCode(code string) (res tables.CouponInfo, err error) {
	code, err = encrypt.AesEncrypt(code, config.Cfg.Das.Coupon.EncryptionKey)
	if err != nil {
		return
	}
	err = d.db.Where("code = ?", code).First(&res).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
		return
	}
	return
}

func (d *DbDao) GetSetInfoByCoupon(coupon string) (res tables.CouponSetInfo, err error) {
	code, err := encrypt.AesEncrypt(coupon, config.Cfg.Das.Coupon.EncryptionKey)
	if err != nil {
		return
	}

	couponInfo := &tables.CouponInfo{}
	if err = d.db.Where("code = ?", code).First(couponInfo).Error; err != nil {
		return
	}
	err = d.db.Where("cid = ?", couponInfo.Cid).First(&res).Error
	return
}

func (d *DbDao) UpdateCouponInfo(cid string, ids []int64, status int) error {
	return d.db.Where("cid=? and status!=? and id in(?)", cid, tables.CouponStatusUsed, ids).Updates(map[string]interface{}{"status": status}).Error
}
