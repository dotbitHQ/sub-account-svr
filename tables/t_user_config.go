package tables

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/shopspring/decimal"
	"time"
)

type UserConfig struct {
	Id            int64          `gorm:"column:id;AUTO_INCREMENT" json:"id"`
	Account       string         `gorm:"column:account;type:varchar(255);comment:父账号;NOT NULL" json:"account"`
	AccountId     string         `gorm:"column:account_id; uniqueIndex:uk_account_id;type:varchar(255);comment:父账号id;NOT NULL" json:"account_id"`
	MintConfig    *MintConfig    `gorm:"column:mint_config;type:text;comment:mint设置内容" json:"mint_config"`
	PaymentConfig *PaymentConfig `gorm:"column:payment_config;type:text;comment:用户收款配置" json:"payment_config"`
	CreatedAt     time.Time      `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

type MintConfig struct {
	Title           string `json:"title"`
	Desc            string `json:"desc"`
	Benefits        string `json:"benefits"`
	Links           []Link `json:"links"`
	BackgroundColor string `json:"background_color"`
	MintSuccessPage []struct {
		Type string `json:"type"`
		Url  string `json:"url"`
	} `json:"mint_success_page"`
}

type Link struct {
	App  string `json:"app"`
	Link string `json:"link"`
}

type PaymentConfig struct {
	CfgMap map[string]PaymentConfigElement `json:"cfg_map"`
}

type PaymentConfigElement struct {
	Enable     bool            `json:"enable"`
	TokenID    string          `json:"token_id"`
	Symbol     string          `json:"symbol"`
	HaveRecord bool            `json:"have_record"`
	Price      decimal.Decimal `json:"price"`
	Decimals   int32           `json:"decimals"`
}

func (m *UserConfig) TableName() string {
	return "t_user_config"
}

func (u *MintConfig) Value() (driver.Value, error) {
	if u == nil {
		return nil, nil
	}
	marshal, _ := json.Marshal(u)
	if string(marshal) == "{}" {
		return nil, nil
	}
	return marshal, nil
}

func (u *MintConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	err := json.Unmarshal(src.([]byte), u)
	if err != nil {
		return err
	}
	return nil
}

func (u *PaymentConfig) Value() (driver.Value, error) {
	if u == nil {
		return nil, nil
	}
	marshal, _ := json.Marshal(u)
	if string(marshal) == "{}" {
		return nil, nil
	}
	return marshal, nil
}

func (u *PaymentConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	err := json.Unmarshal(src.([]byte), u)
	if err != nil {
		return err
	}
	return nil
}
