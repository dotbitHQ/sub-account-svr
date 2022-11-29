package main

import (
	"context"
	"das_sub_account/block_parser"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/http_server"
	"das_sub_account/http_server/handle"
	"das_sub_account/task"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/dascache"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/mylog"
	"github.com/scorpiotzh/toolib"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"sync"
	"time"
)

var (
	log               = mylog.NewLogger("main", mylog.LevelDebug)
	exit              = make(chan struct{})
	ctxServer, cancel = context.WithCancel(context.Background())
	wgServer          = sync.WaitGroup{}
)

func main() {
	log.Debugf("start：")
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(ctx *cli.Context) error {
	// config file
	configFilePath := ctx.String("config")
	if err := config.InitCfg(configFilePath); err != nil {
		return err
	}

	// config file watcher
	watcher, err := config.AddCfgFileWatcher(configFilePath)
	if err != nil {
		return fmt.Errorf("AddCfgFileWatcher err: %s", err.Error())
	}
	// ============= service start =============

	// das core
	dasCore, dasCache, err := initDasCore()
	if err != nil {
		return fmt.Errorf("initDasCore err: %s", err.Error())
	}
	log.Infof("das core ok")

	// tx builder
	txBuilderBase, serverScript, err := initTxBuilder(dasCore)
	if err != nil {
		return fmt.Errorf("initTxBuilder err: %s", err.Error())
	}
	log.Infof("tx builder ok")

	// db
	dbDao, err := dao.NewGormDB(config.Cfg.DB.Mysql, config.Cfg.DB.ParserMysql, config.Cfg.Slb.SvrName == "")
	if err != nil {
		return fmt.Errorf("NewGormDB err: %s", err.Error())
	}
	log.Infof("db ok")

	// redis
	red, err := toolib.NewRedisClient(config.Cfg.Cache.Redis.Addr, config.Cfg.Cache.Redis.Password, config.Cfg.Cache.Redis.DbNum)
	if err != nil {
		return fmt.Errorf("NewRedisClient err:%s", err.Error())
	} else {
		log.Info("redis ok")
	}
	rc := &cache.RedisCache{
		Ctx: ctxServer,
		Red: red,
	}

	// mongo
	mongoClient, err := mongo.Connect(ctxServer, options.Client().ApplyURI(config.Cfg.DB.Mongo.Uri))
	if err != nil {
		return fmt.Errorf("mongo.Connect err:%s", err.Error())
	}
	log.Infof("mongo ok")

	// tx tool
	txTool := &txtool.SubAccountTxTool{
		Ctx:           ctxServer,
		Mongo:         mongoClient,
		DbDao:         dbDao,
		DasCore:       dasCore,
		DasCache:      dasCache,
		ServerScript:  serverScript,
		TxBuilderBase: txBuilderBase,
	}
	log.Infof("tx tool ok")

	// block parser
	if config.Cfg.Slb.SvrName == "" {
		blockParser := block_parser.BlockParser{
			DasCore:            dasCore,
			CurrentBlockNumber: config.Cfg.Chain.CurrentBlockNumber,
			DbDao:              dbDao,
			ConcurrencyNum:     config.Cfg.Chain.ConcurrencyNum,
			ConfirmNum:         config.Cfg.Chain.ConfirmNum,
			Mongo:              mongoClient,
			Ctx:                ctxServer,
			Wg:                 &wgServer,
		}
		if err := blockParser.Run(); err != nil {
			return fmt.Errorf("blockParser.Run() err: %s", err.Error())
		}
		log.Infof("block parser ok")
	}

	// task
	smtTask := task.SmtTask{
		Ctx:      ctxServer,
		Wg:       &wgServer,
		DbDao:    dbDao,
		DasCore:  dasCore,
		Mongo:    mongoClient,
		TxTool:   txTool,
		RC:       rc,
		MaxRetry: config.Cfg.Das.MaxRetry,
	}
	smtTask.RunTaskCheckTx()
	smtTask.RunTaskConfirmOtherTx()
	//smtTask.RunCheckError()
	smtTask.RunTaskRollback()
	//smtTask.RunTaskDistribution()
	//smtTask.RunMintTaskDistribution()
	smtTask.RunUpdateSubAccountTaskDistribution()
	//smtTask.RunEditSubAccountTask()
	//smtTask.RunCreateSubAccountTask()
	smtTask.RunUpdateSubAccountTask()
	log.Infof("task ok")

	// http
	hs := http_server.HttpServer{
		Ctx:             ctxServer,
		Address:         config.Cfg.Server.HttpServerAddr,
		InternalAddress: config.Cfg.Server.HttpServerInternalAddr,
		H: &handle.HttpHandle{
			Ctx:           ctxServer,
			DasCore:       dasCore,
			DasCache:      dasCache,
			TxBuilderBase: txBuilderBase,
			DbDao:         dbDao,
			RC:            rc,
			TxTool:        txTool,
			Mongo:         mongoClient,
		},
	}
	hs.Run()
	log.Info("http server ok")
	// ============= service end =============
	toolib.ExitMonitoring(func(sig os.Signal) {
		log.Warn("ExitMonitoring:", sig.String())
		if watcher != nil {
			log.Warn("close watcher ... ")
			_ = watcher.Close()
		}
		cancel()
		wgServer.Wait()
		log.Warn("success exit server. bye bye!")
		time.Sleep(time.Second)
		exit <- struct{}{}
	})

	<-exit

	return nil
}

func initDasCore() (*core.DasCore, *dascache.DasCache, error) {
	// ckb node
	ckbClient, err := rpc.DialWithIndexer(config.Cfg.Chain.CkbUrl, config.Cfg.Chain.IndexUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("rpc.DialWithIndexer err: %s", err.Error())
	}
	log.Info("ckb node ok")

	env := core.InitEnvOpt(config.Cfg.Server.Net,
		common.DasContractNameConfigCellType,
		common.DasContractNameAccountCellType,
		common.DasContractNameBalanceCellType,
		common.DasContractNameDispatchCellType,
		common.DasContractNameAlwaysSuccess,
		common.DASContractNameSubAccountCellType,
		common.DASContractNameEip712LibCellType,
	)

	// das init
	ops := []core.DasCoreOption{
		core.WithClient(ckbClient),
		core.WithDasContractArgs(env.ContractArgs),
		core.WithDasContractCodeHash(env.ContractCodeHash),
		core.WithDasNetType(config.Cfg.Server.Net),
		core.WithTHQCodeHash(env.THQCodeHash),
	}
	dasCore := core.NewDasCore(ctxServer, &wgServer, ops...)
	dasCore.InitDasContract(env.MapContract)
	if err := dasCore.InitDasConfigCell(); err != nil {
		return nil, nil, fmt.Errorf("InitDasConfigCell err: %s", err.Error())
	}
	if err := dasCore.InitDasSoScript(); err != nil {
		return nil, nil, fmt.Errorf("InitDasSoScript err: %s", err.Error())
	}
	dasCore.RunAsyncDasContract(time.Minute * 3)   // contract outpoint
	dasCore.RunAsyncDasConfigCell(time.Minute * 4) // config cell outpoint
	dasCore.RunAsyncDasSoScript(time.Minute * 5)   // so

	log.Info("das contract ok")

	// das cache
	dasCache := dascache.NewDasCache(ctxServer, &wgServer)
	dasCache.RunClearExpiredOutPoint(time.Minute * 5)
	log.Info("das cache ok")

	return dasCore, dasCache, nil
}

func initTxBuilder(dasCore *core.DasCore) (*txbuilder.DasTxBuilderBase, *types.Script, error) {
	serverAddressScriptArgs := ""
	var serverScript *types.Script
	if config.Cfg.Server.ServerAddress != "" {
		parseAddress, err := address.Parse(config.Cfg.Server.ServerAddress)
		if err != nil {
			return nil, nil, fmt.Errorf("server address.Parse err: %s", err.Error())
		} else {
			serverAddressScriptArgs = common.Bytes2Hex(parseAddress.Script.Args)
			serverScript = parseAddress.Script
		}
	}

	var handleSign sign.HandleSignCkbMessage
	if config.Cfg.Server.RemoteSignApiUrl != "" && serverAddressScriptArgs != "" {
		remoteSignClient, err := sign.NewClient(ctxServer, config.Cfg.Server.RemoteSignApiUrl)
		if err != nil {
			return nil, nil, fmt.Errorf("sign.NewClient err: %s", err.Error())
		}
		handleSign = sign.RemoteSign(remoteSignClient, config.Cfg.Server.Net, serverAddressScriptArgs)
	} else if config.Cfg.Server.ServerPrivateKey != "" {
		handleSign = sign.LocalSign(config.Cfg.Server.ServerPrivateKey)
	}

	txBuilderBase := txbuilder.NewDasTxBuilderBase(ctxServer, dasCore, handleSign, serverAddressScriptArgs)

	return txBuilderBase, serverScript, nil
}
