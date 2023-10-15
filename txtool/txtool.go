package txtool

import (
	"context"
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/dascache"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"net"
	"sync"
	"time"
)

var (
	log          = logger.NewLogger("txtool", logger.LevelDebug)
	PromRegister = prometheus.NewRegistry()
)

var Tools *SubAccountTxTool

type Metric struct {
	l         sync.Mutex
	api       *prometheus.SummaryVec
	errNotify *prometheus.CounterVec
}

func (m *Metric) Api() *prometheus.SummaryVec {
	if m.api == nil {
		m.l.Lock()
		defer m.l.Unlock()
		m.api = prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name: "api",
		}, []string{"method", "http_status", "err_no", "err_msg"})
		PromRegister.MustRegister(m.api)
	}
	return m.api
}

func (m *Metric) ErrNotify() *prometheus.CounterVec {
	if m.errNotify == nil {
		m.l.Lock()
		defer m.l.Unlock()
		m.errNotify = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notify",
		}, []string{"title", "text"})
		PromRegister.MustRegister(m.errNotify)
	}
	return m.errNotify
}

func Init(params *SubAccountTxTool) {
	Tools = params
}

type SubAccountTxTool struct {
	Ctx           context.Context
	DbDao         *dao.DbDao
	DasCore       *core.DasCore
	DasCache      *dascache.DasCache
	ServerScript  *types.Script
	TxBuilderBase *txbuilder.DasTxBuilderBase
	Pusher        *push.Pusher
	Metrics       Metric
}

type ParamBuildTxs struct {
	TaskList             []tables.TableTaskInfo
	TaskMap              map[string][]tables.TableSmtRecordInfo
	Account              *tables.TableAccountInfo // parent account
	SubAccountLiveCell   *indexer.LiveCell
	Tree                 *smt.SmtServer
	BaseInfo             *BaseInfo
	BalanceDasLock       *types.Script
	BalanceDasType       *types.Script
	SubAccountIds        []string
	SubAccountValueMap   map[string]string
	SubAccountBuilderMap map[string]*witness.SubAccountNew
}

type ResultBuildTxs struct {
	//IsCustomScript   bool
	DasTxBuilderList []*txbuilder.DasTxBuilder
}

func (s *SubAccountTxTool) Run() {
	if config.Cfg.Server.PrometheusPushGateway != "" && config.Cfg.Server.Name != "" {
		s.Pusher = push.New(config.Cfg.Server.PrometheusPushGateway, config.Cfg.Server.Name)
		s.Pusher.Gatherer(PromRegister)
		s.Pusher.Grouping("env", fmt.Sprint(config.Cfg.Server.Net))
		s.Pusher.Grouping("instance", GetLocalIp("eth0"))

		go func() {
			ticker := time.NewTicker(time.Second * 5)
			defer ticker.Stop()

			for range ticker.C {
				_ = s.Pusher.Push()
			}
		}()
	}
}

func (s *SubAccountTxTool) BuildTxsForUpdateSubAccount(p *ParamBuildTxs) (*ResultBuildTxs, error) {
	var res ResultBuildTxs

	newSubAccountPrice, _ := molecule.Bytes2GoU64(p.BaseInfo.ConfigCellBuilder.ConfigCellSubAccount.NewSubAccountPrice().RawData())
	renewSubAccountPrice, _ := p.BaseInfo.ConfigCellBuilder.RenewSubAccountPrice()
	commonFee, _ := molecule.Bytes2GoU64(p.BaseInfo.ConfigCellBuilder.ConfigCellSubAccount.CommonFee().RawData())

	// outpoint
	accountOutPoint := common.String2OutPointStruct(p.Account.Outpoint)
	subAccountOutpoint := p.SubAccountLiveCell.OutPoint

	// account
	//res.IsCustomScript = s.isCustomScript(p.SubAccountLiveCell.OutputData)
	subAccountCellOutput := p.SubAccountLiveCell.Output
	subAccountOutputsData := p.SubAccountLiveCell.OutputData
	// build txs
	for i, task := range p.TaskList {
		records, ok := p.TaskMap[task.TaskId]
		if !ok {
			continue
		}
		switch task.Action {
		case common.DasActionUpdateSubAccount:
			var resUpdate *ResultBuildUpdateSubAccountTx
			var err error
			//if res.IsCustomScript {
			//	resUpdate, err = s.BuildUpdateSubAccountTxForCustomScript(&ParamBuildUpdateSubAccountTx{
			//		TaskInfo:              &p.TaskList[i],
			//		Account:               p.Account,
			//		AccountOutPoint:       accountOutPoint,
			//		SubAccountOutpoint:    subAccountOutpoint,
			//		SmtRecordInfoList:     records,
			//		Tree:                  p.Tree,
			//		BaseInfo:              p.BaseInfo,
			//		SubAccountBuilderMap:  p.SubAccountBuilderMap,
			//		NewSubAccountPrice:    newSubAccountPrice,
			//		RenewSubAccountPrice:  renewSubAccountPrice,
			//		BalanceDasLock:        p.BalanceDasLock,
			//		BalanceDasType:        p.BalanceDasType,
			//		CommonFee:             commonFee,
			//		SubAccountCellOutput:  subAccountCellOutput,
			//		SubAccountOutputsData: subAccountOutputsData,
			//	})
			//	if err != nil {
			//		return nil, fmt.Errorf("BuildUpdateSubAccountTxForCustomScript err: %s", err.Error())
			//	}
			//} else {
			resUpdate, err = s.BuildUpdateSubAccountTx(&ParamBuildUpdateSubAccountTx{
				TaskInfo:              &p.TaskList[i],
				Account:               p.Account,
				AccountOutPoint:       accountOutPoint,
				SubAccountOutpoint:    subAccountOutpoint,
				SmtRecordInfoList:     records,
				Tree:                  p.Tree,
				BaseInfo:              p.BaseInfo,
				SubAccountBuilderMap:  p.SubAccountBuilderMap,
				NewSubAccountPrice:    newSubAccountPrice,
				RenewSubAccountPrice:  renewSubAccountPrice,
				BalanceDasLock:        p.BalanceDasLock,
				BalanceDasType:        p.BalanceDasType,
				CommonFee:             commonFee,
				SubAccountCellOutput:  subAccountCellOutput,
				SubAccountOutputsData: subAccountOutputsData,
			})
			if err != nil {
				return nil, fmt.Errorf("BuildUpdateSubAccountTx err: %s", err.Error())
			}
			//	}
			res.DasTxBuilderList = append(res.DasTxBuilderList, resUpdate.DasTxBuilder)
		default:
			return nil, fmt.Errorf("not exist action [%s]", task.Action)
		}
	}

	return &res, nil
}

func GetLocalIp(interfaceName string) string {
	ief, err := net.InterfaceByName(interfaceName)
	if err != nil {
		log.Error("GetLocalIp: ", err)
		return ""
	}
	addrs, err := ief.Addrs()
	if err != nil {
		log.Error("GetLocalIp: ", err)
		return ""
	}

	var ipv4Addr net.IP
	for _, addr := range addrs {
		if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
			break
		}
	}
	if ipv4Addr == nil {
		log.Errorf("GetLocalIp interface %s don't have an ipv4 address", interfaceName)
		return ""
	}
	return ipv4Addr.String()
}
