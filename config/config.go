package config

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/fsnotify/fsnotify"
	"github.com/scorpiotzh/mylog"
	"github.com/scorpiotzh/toolib"
)

var (
	Cfg CfgServer
	log = mylog.NewLogger("config", mylog.LevelDebug)
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
			ServiceFeeRatio     float64           `json:"service_fee_ratio" yaml:"service_fee_ratio"`
		} `json:"auto_mint" yaml:"auto_mint"`
	} `json:"das" yaml:"das"`
	Origins []string `json:"origins" yaml:"origins"`
	Notify  struct {
		LarkErrorKey               string `json:"lark_error_key" yaml:"lark_error_key"`
		LarkCreateSubAccountKey    string `json:"lark_create_sub_account_key" yaml:"lark_create_sub_account_key"`
		DiscordCreateSubAccountKey string `json:"discord_create_sub_account_key" yaml:"discord_create_sub_account_key"`
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
		Mongo       struct {
			Uri         string `json:"uri" yaml:"uri"`
			SmtDatabase string `json:"smt_database" yaml:"smt_database"`
		} `json:"mongo" yaml:"mongo"`
	} `json:"db" yaml:"db"`
	Cache struct {
		Redis struct {
			Addr     string `json:"addr" yaml:"addr"`
			Password string `json:"password" yaml:"password"`
			DbNum    int    `json:"db_num" yaml:"db_num"`
		} `json:"redis" yaml:"redis"`
	} `json:"cache" yaml:"cache"`
	SuspendMap map[string]string `json:"suspend_map" yaml:"suspend_map"`
}

type Server struct {
	Name   string `json:"name" yaml:"name"`
	Url    string `json:"url" yaml:"url"`
	Weight int    `json:"weight" yaml:"weight"`
}
