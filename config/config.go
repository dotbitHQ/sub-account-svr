package config

import (
	"context"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"github.com/sjatsh/uint128"
)

var (
	Cfg CfgServer
	log = logger.NewLogger("config", logger.LevelDebug)
)

func InitCfg(configFilePath string) error {
	if configFilePath == "" {
		configFilePath = "./config/config.yaml"
	}
	log.Info("config file：", configFilePath)
	if err := toolib.UnmarshalYamlFile(configFilePath, &Cfg); err != nil {
		return fmt.Errorf("UnmarshalYamlFile err:%s", err.Error())
	}
	log.Info("config file：ok")
	//log.Info("config file：", toolib.JsonString(Cfg))
	return nil
}

func AddCfgFileWatcher(configFilePath string) (*fsnotify.Watcher, error) {
	if configFilePath == "" {
		configFilePath = "./config/config.yaml"
	}
	return toolib.AddFileWatcher(configFilePath, func() {
		log.Info("update config file：", configFilePath)
		if err := toolib.UnmarshalYamlFile(configFilePath, &Cfg); err != nil {
			log.Error("UnmarshalYamlFile err:", err.Error())
		}
		log.Info("update config file：ok")
		//log.Info("update config file：", toolib.JsonString(Cfg))
	})
}

type DbMysql struct {
	Addr        string `json:"addr" yaml:"addr"`
	User        string `json:"user" yaml:"user"`
	Password    string `json:"password" yaml:"password"`
	DbName      string `json:"db_name" yaml:"db_name"`
	MaxOpenConn int    `json:"max_open_conn" yaml:"max_open_conn"`
	MaxIdleConn int    `json:"max_idle_conn" yaml:"max_idle_conn"`
}

type CfgServer struct {
	Slb struct {
		SvrName string   `json:"svr_name" yaml:"svr_name"`
		Servers []Server `json:"servers" yaml:"servers"`
	} `json:"slb" yaml:"slb"`
	Server struct {
		Name                   string            `json:"name" yaml:"name"`
		IsUpdate               bool              `json:"is_update" yaml:"is_update"`
		Net                    common.DasNetType `json:"net" yaml:"net"`
		HttpServerAddr         string            `json:"http_server_addr" yaml:"http_server_addr"`
		HttpServerInternalAddr string            `json:"http_server_internal_addr" yaml:"http_server_internal_addr"`
		ParserUrl              string            `json:"parser_url" yaml:"parser_url"`
		ServerAddress          string            `json:"server_address" yaml:"server_address"`
		ServerPrivateKey       string            `json:"server_private_key" yaml:"server_private_key"`
		RemoteSignApiUrl       string            `json:"remote_sign_api_url" yaml:"remote_sign_api_url"`
		PushLogUrl             string            `json:"push_log_url" yaml:"push_log_url"`
		PushLogIndex           string            `json:"push_log_index" yaml:"push_log_index"`
		NotExit                bool              `json:"not_exit" yaml:"not_exit"`
		SmtServer              string            `json:"smt_server" yaml:"smt_server"`
		UniPayUrl              string            `json:"uni_pay_url" yaml:"uni_pay_url"`
		RefundSwitch           bool              `json:"refund_switch" yaml:"refund_switch"`
		RecycleSwitch          bool              `json:"recycle_switch" yaml:"recycle_switch"`
		RecycleLimit           int               `json:"recycle_limit" yaml:"recycle_limit"`
		PrometheusPushGateway  string            `json:"prometheus_push_gateway" yaml:"prometheus_push_gateway"`
		TxTeeRate              uint64            `json:"tx_fee_rate" yaml:"tx_fee_rate"`
	} `json:"server" yaml:"server"`
	Das struct {
		MaxRegisterYears uint64 `json:"max_register_years" yaml:"max_register_years"`
		MaxCreateCount   int    `json:"max_create_count" yaml:"max_create_count"`
		MaxUpdateCount   int    `json:"max_update_count" yaml:"max_update_count"`
		MaxRetry         int    `json:"max_retry" yaml:"max_retry"`
		AutoMint         struct {
			SupportPaymentToken []string          `json:"support_payment_token" yaml:"support_payment_token"`
			BackgroundColors    map[string]string `json:"background_colors" yaml:"background_colors"`
			PaymentMinPrice     int64             `json:"payment_min_price" yaml:"payment_min_price"`
			ServiceFeeRatio     string            `json:"service_fee_ratio" yaml:"service_fee_ratio"`
			PlatformFeeRatio    string            `json:"platform_fee_ratio" yaml:"platform_fee_ratio"`
			ServiceFeeMin       float64           `json:"service_fee_min" yaml:"service_fee_min"`
			MinPrice            string            `json:"min_price" yaml:"min_price"`
		} `json:"auto_mint" yaml:"auto_mint"`
		Approval struct {
			MaxDelayCount uint8 `json:"max_delay_count" yaml:"max_delay_count"`
		} `json:"approval" yaml:"approval"`
		Coupon struct {
			CouponPrice   float64 `json:"coupon_price" yaml:"coupon_price"`
			EncryptionKey string  `json:"encryption_key" yaml:"encryption_key"`
			PriceMin      float64 `json:"price_min" yaml:"price_min"`
			PriceMax      float64 `json:"price_max" yaml:"price_max"`
		} `json:"coupon" yaml:"coupon"`
		JwtKey string `json:"jwt_key" yaml:"jwt_key"`
		Dp     struct {
			TransferWhiteList string `json:"transfer_white_list" yaml:"transfer_white_list"`
			CapacityWhitelist string `json:"capacity_whitelist" yaml:"capacity_whitelist"`
			TimeOnline        int64  `json:"time_online" yaml:"time_online"`
		} `json:"dp" yaml:"dp"`
	} `json:"das" yaml:"das"`
	Origins []string `json:"origins" yaml:"origins"`
	Notify  struct {
		LarkCreateSubAccountKey     string `json:"lark_create_sub_account_key" yaml:"lark_create_sub_account_key"`
		DiscordCreateSubAccountKey  string `json:"discord_create_sub_account_key" yaml:"discord_create_sub_account_key"`
		LarkParentAccountPaymentKey string `json:"lark_parent_account_payment_key" yaml:"lark_parent_account_payment_key"`
		SentryDsn                   string `json:"sentry_dsn" yaml:"sentry_dsn"`
	} `json:"notify" yaml:"notify"`
	Chain struct {
		CkbUrl             string `json:"ckb_url" yaml:"ckb_url"`
		IndexUrl           string `json:"index_url" yaml:"index_url"`
		CurrentBlockNumber uint64 `json:"current_block_number" yaml:"current_block_number"`
		ConfirmNum         uint64 `json:"confirm_num" yaml:"confirm_num"`
		ConcurrencyNum     uint64 `json:"concurrency_num" yaml:"concurrency_num"`
	} `json:"chain" yaml:"chain"`
	DB struct {
		Mysql       DbMysql `json:"mysql" yaml:"mysql"`
		ParserMysql DbMysql `json:"parser_mysql" yaml:"parser_mysql"`
	} `json:"db" yaml:"db"`
	Cache struct {
		Redis struct {
			Addr     string `json:"addr" yaml:"addr"`
			Password string `json:"password" yaml:"password"`
			DbNum    int    `json:"db_num" yaml:"db_num"`
		} `json:"redis" yaml:"redis"`
	} `json:"cache" yaml:"cache"`
	SuspendMap       map[string]string `json:"suspend_map" yaml:"suspend_map"`
	UnipayAddressMap map[string]string `json:"unipay_address_map" yaml:"unipay_address_map"`
	Stripe           struct {
		PremiumPercentage decimal.Decimal `json:"premium_percentage" yaml:"premium_percentage"`
		PremiumBase       decimal.Decimal `json:"premium_base" yaml:"premium_base"`
	} `json:"stripe" yaml:"stripe"`
}

type Server struct {
	Name   string `json:"name" yaml:"name"`
	Url    string `json:"url" yaml:"url"`
	Weight int    `json:"weight" yaml:"weight"`
}

func GetUnipayAddress(tokenId tables.TokenId) string {
	switch tokenId {
	case tables.TokenIdEth, tables.TokenIdErc20USDT,
		tables.TokenIdBnb, tables.TokenIdBep20USDT,
		tables.TokenIdPOL:
		//tables.TokenIdMatic:
		return Cfg.UnipayAddressMap["evm"]
	case tables.TokenIdTrx, tables.TokenIdTrc20USDT:
		return Cfg.UnipayAddressMap["tron"]
	case tables.TokenIdCkb, tables.TokenIdCkbDas:
		return Cfg.UnipayAddressMap["ckb"]
	}
	return ""
}

func PriceToCKB(ctx context.Context, price, quote, years uint64) (total uint64) {
	log.Info(ctx, "PriceToCKB:", price, quote, years)
	if quote == 0 {
		return 0
	}
	total = uint128.From64(price).Mul(uint128.From64(common.OneCkb)).Div(uint128.From64(quote)).Mul(uint128.From64(years)).Big().Uint64()
	log.Info(ctx, "PriceToCKB:", price, quote, total)
	return
}
