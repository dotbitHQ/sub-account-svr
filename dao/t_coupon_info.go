package dao

import (
	"das_sub_account/tables"
	"errors"
	"gorm.io/gorm"
)

func (d *DbDao) CouponExists(codes []string) (map[string]bool, error) {
	find := make([]*tables.CouponInfo, 0)
	if err := d.db.Where("code in (?)", codes).Find(&find).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	res := make(map[string]bool)
	for _, v := range find {
		res[v.Code] = true
	}
	return res, nil
}

func (d *DbDao) CreateCoupon(set *tables.CouponSetInfo, codes []*tables.CouponInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(set).Error; err != nil {
			return err
		}
		for _, v := range codes {
			if err := tx.Create(v).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
