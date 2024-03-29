package dao

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/scorpiotzh/toolib"
	"gorm.io/gorm"
)

type DbDao struct {
	db       *gorm.DB
	parserDb *gorm.DB
}

var (
	log = logger.NewLogger("dao", logger.LevelDebug)
)

func NewGormDB(dbMysql, parserMysql config.DbMysql, autoMigrate bool) (*DbDao, error) {
	db, err := http_api.NewGormDB(dbMysql.Addr, dbMysql.User, dbMysql.Password, dbMysql.DbName, dbMysql.MaxOpenConn, dbMysql.MaxIdleConn)
	if err != nil {
		return nil, fmt.Errorf("toolib.NewGormDB err: %s", err.Error())
	}

	if db.Migrator().HasIndex(&tables.TableSmtRecordInfo{}, "uk_account_nonce") {
		log.Info("HasIndex: uk_account_nonce")
		if err := db.Migrator().DropIndex(&tables.TableSmtRecordInfo{}, "uk_account_nonce"); err != nil {
			return nil, fmt.Errorf("DropIndex err: %s", err.Error())
		}
	}

	// AutoMigrate will create tables, missing foreign keys, constraints, columns and indexes.
	// It will change existing column’s type if its size, precision, nullable changed.
	// It WON’T delete unused columns to protect your data.
	if autoMigrate {
		if err = db.AutoMigrate(
			&tables.TableBlockParserInfo{},
			&tables.TableSmtRecordInfo{},
			&tables.TableTaskInfo{},
			&tables.TableMintSignInfo{},
			&tables.AutoPaymentInfo{},
			&tables.OrderInfo{},
			&tables.PaymentInfo{},
			&tables.UserConfig{},
			&tables.RuleWhitelist{},
			&tables.TableSubAccountAutoMintWithdrawHistory{},
			&tables.CouponSetInfo{},
			&tables.CouponInfo{},
			&tables.TablePendingInfo{},
		); err != nil {
			return nil, err
		}

		if err := db.Migrator().AlterColumn(&tables.TableMintSignInfo{}, "key_value"); err != nil {
			return nil, fmt.Errorf("AlterColumn err: %s", err.Error())
		}

	}

	parserDb, err := http_api.NewGormDB(parserMysql.Addr, parserMysql.User, parserMysql.Password, parserMysql.DbName, parserMysql.MaxOpenConn, parserMysql.MaxIdleConn)
	if err != nil {
		return nil, fmt.Errorf("toolib.NewGormDB err: %s", err.Error())
	}
	return &DbDao{db: db, parserDb: parserDb}, nil
}

func (d *DbDao) Transaction(fc func(tx *gorm.DB) error) error {
	return d.db.Transaction(fc)
}

func NewDbDao(dbMysql, parserMysql config.DbMysql) (*DbDao, error) {
	db, err := toolib.NewGormDB(dbMysql.Addr, dbMysql.User, dbMysql.Password, dbMysql.DbName, dbMysql.MaxOpenConn, dbMysql.MaxIdleConn)
	if err != nil {
		return nil, fmt.Errorf("toolib.NewGormDB err: %s", err.Error())
	}
	parserDb, err := toolib.NewGormDB(parserMysql.Addr, parserMysql.User, parserMysql.Password, parserMysql.DbName, parserMysql.MaxOpenConn, parserMysql.MaxIdleConn)
	if err != nil {
		return nil, fmt.Errorf("toolib.NewGormDB err: %s", err.Error())
	}
	return &DbDao{db: db, parserDb: parserDb}, nil
}
