package tables

import (
	"time"
)

// TTokenPriceInfo 代币价格信息表
type TTokenPriceInfo struct {
	Id            uint64    `gorm:"column:id;AUTO_INCREMENT;comment:自增主键" json:"id"`
	TokenId       string    `gorm:"column:token_id;type:varchar(255);NOT NULL" json:"token_id"`
	GeckoId       string    `gorm:"column:gecko_id;type:varchar(255);comment:the id from coingecko;NOT NULL" json:"gecko_id"`
	ChainType     int       `gorm:"column:chain_type;type:smallint(6);default:0;NOT NULL" json:"chain_type"`
	Contract      string    `gorm:"column:contract;type:varchar(255);NOT NULL" json:"contract"`
	Name          string    `gorm:"column:name;type:varchar(255);comment:the name of token;NOT NULL" json:"name"`
	Symbol        string    `gorm:"column:symbol;type:varchar(255);comment:the symbol of token;NOT NULL" json:"symbol"`
	Decimals      int       `gorm:"column:decimals;type:smallint(6);default:0;NOT NULL" json:"decimals"`
	Price         float64   `gorm:"column:price;type:decimal(50,8);default:0.00000000;NOT NULL" json:"price"`
	Logo          string    `gorm:"column:logo;type:varchar(255);NOT NULL" json:"logo"`
	Change24H     float64   `gorm:"column:change_24_h;type:decimal(50,8);default:0.00000000;NOT NULL" json:"change_24_h"`
	Vol24H        float64   `gorm:"column:vol_24_h;type:decimal(50,8);default:0.00000000;NOT NULL" json:"vol_24_h"`
	MarketCap     float64   `gorm:"column:market_cap;type:decimal(50,8);default:0.00000000;NOT NULL" json:"market_cap"`
	LastUpdatedAt uint64    `gorm:"column:last_updated_at;type:bigint(20) unsigned;default:0;NOT NULL" json:"last_updated_at"`
	Status        int       `gorm:"column:status;type:smallint(6);default:0;comment:0: normal 1: banned;NOT NULL" json:"status"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

func (m *TTokenPriceInfo) TableName() string {
	return "t_token_price_info"
}
