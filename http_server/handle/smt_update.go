package handle

import (
	"bytes"
	"context"
	"das_sub_account/cache"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"sync"
)

const (
	//The kv limit number One of one rpc request to smt server,t's up to your smt service
	SyncTreeLimit = 3000
)

type ReqSmtUpdate struct {
	ParentAccountId string `json:"parent_account_id"`
	SubAccountId    string `json:"sub_account_id"`
	Value           string `json:"value"`
}
type ReqSmtSync struct {
	AccountIds []string `json:"account_ids""`
}

type RespSmtUpdate struct {
	Root string `json:"root"`
}
type RespSmtSync struct {
	SyncFaildAcc []string
}

func (h *HttpHandle) SmtSync(ctx *gin.Context) {
	var (
		funcName = "SmtSync"
		clientIp = GetClientIp(ctx)
		req      ReqSmtSync
		apiResp  api_code.ApiResp
		err      error
	)
	if err := ctx.BindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))
	if err = h.doSmtSync(&req, &apiResp); err != nil {
		log.Error("doSmtSync err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) SmtUpdate(ctx *gin.Context) {
	var (
		funcName = "SmtUpdate"
		clientIp = GetClientIp(ctx)
		req      ReqSmtUpdate
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doSmtUpdate(&req, &apiResp); err != nil {
		log.Error("doSmtUpdate err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSmtUpdate(req *ReqSmtUpdate, apiResp *api_code.ApiResp) error {
	var resp RespSmtUpdate

	if req.ParentAccountId == "" || req.SubAccountId == "" || req.Value == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}
	log.Info("doSmtUpdate:", req.ParentAccountId, req.SubAccountId, req.Value)

	parentAccountId := req.ParentAccountId
	// do check
	resCheck, err := h.TxTool.DoCheckBeforeBuildTx(parentAccountId)
	if err != nil {
		if resCheck != nil && resCheck.Continue {
			apiResp.ApiRespErr(api_code.ApiCodeTaskInProgress, "task in progress")
			return nil
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("DoCheckBeforeBuildTx err: %s", err.Error())
	}

	// lock smt and unlock
	if err := h.RC.LockWithRedis(parentAccountId); err != nil {
		if err == cache.ErrDistributedLockPreemption {
			apiResp.ApiRespErr(api_code.ApiCodeDistributedLockPreemption, err.Error())
		} else {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		}
		return fmt.Errorf("LockWithRedis err: %s", err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := h.RC.UnLockWithRedis(parentAccountId); err != nil {
			fmt.Println("UnLockWithRedis:", err.Error())
		}
		cancel()
	}()
	h.RC.DoLockExpire(ctx, parentAccountId)

	// get smt tree
	tree := smt.NewSmtSrv(*h.SmtServerUrl, parentAccountId)
	key := smt.AccountIdToSmtH256(req.SubAccountId)
	value := common.Hex2Bytes(req.Value)
	var kv []smt.SmtKv
	kv = append(kv, smt.SmtKv{
		key,
		value,
	})
	opt := smt.SmtOpt{
		GetRoot:  true,
		GetProof: false,
	}
	r, err := tree.UpdateSmt(kv, opt)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("tree.Update err: %s", err.Error())
	}
	root := r.Root
	resp.Root = root.String()
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doSmtSync(req *ReqSmtSync, apiResp *api_code.ApiResp) error {
	var (
		resp RespSmtSync
		list []tables.TableSmtInfo
		err  error
	)

	if len(req.AccountIds) > 0 {
		list, err = h.DbDao.GetSmtInfoGroupsByAccountIds(req.AccountIds)
	} else {
		list, err = h.DbDao.GetSmtInfoGroups()
	}
	if err != nil {
		return fmt.Errorf("GetParentAccountIds  err: %s", err.Error())
	}

	var chanParentAccountId = make(chan string, 50)
	var faildAcc = sync.Map{}
	var wgTask sync.WaitGroup
	wgTask.Add(1)
	go func() {
		defer wgTask.Done()
		for _, parentAccountId := range list {
			chanParentAccountId <- parentAccountId.ParentAccountId
		}
		close(chanParentAccountId)
	}()

	for i := 0; i < 10; i++ {
		wgTask.Add(1)
		go func() {
			defer wgTask.Done()
		OutLoop:
			for parentAccountId := range chanParentAccountId {
				var opt smt.SmtOpt
				opt.GetRoot = true
				opt.GetProof = false

				tree := smt.NewSmtSrv(*h.SmtServerUrl, parentAccountId)
				smtInfo, err := h.DbDao.GetSmtInfoByParentId(parentAccountId)
				if err != nil {
					log.Warn("GetSmtInfoByParentId err: %s", err.Error())
					faildAcc.Store(parentAccountId, struct{}{})
					continue
				}

				var smtKvTemp []smt.SmtKv
				var currentRoot smt.H256
				for j, _ := range smtInfo {
					if len(smtKvTemp) == SyncTreeLimit {
						res, err := tree.UpdateSmt(smtKvTemp, opt)
						smtKvTemp = []smt.SmtKv{}
						if err != nil {
							log.Warn("tree.Update err: %s", err.Error())
							faildAcc.Store(parentAccountId, struct{}{})
							continue OutLoop
						}
						currentRoot = res.Root
					}

					k := smtInfo[j].AccountId
					v := smtInfo[j].LeafDataHash
					k1 := smt.AccountIdToSmtH256(k)
					var v1 smt.H256
					v1 = common.Hex2Bytes(v)
					smtKvTemp = append(smtKvTemp, smt.SmtKv{
						Key:   k1,
						Value: v1,
					})
				}

				if len(smtKvTemp) > 0 {
					res, err := tree.UpdateSmt(smtKvTemp, opt)
					if err != nil {
						log.Warn("tree.Update err: %s", err.Error())
						faildAcc.Store(parentAccountId, struct{}{})
						continue
					}
					currentRoot = res.Root
				}

				log.Info("sync success : ", parentAccountId)
				contractSubAcc, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
				if err != nil {
					log.Warn("GetDasContractInfo err: %s", err.Error())
					faildAcc.Store(parentAccountId, struct{}{})
					continue
				}
				searchKey := indexer.SearchKey{
					Script:     contractSubAcc.ToScript(common.Hex2Bytes(parentAccountId)),
					ScriptType: indexer.ScriptTypeType,
					ArgsLen:    0,
					Filter:     nil,
				}
				subAccLiveCells, err := h.DasCore.Client().GetCells(h.Ctx, &searchKey, indexer.SearchOrderDesc, 1, "")
				if err != nil {
					log.Warn("GetCells err: %s", err.Error())
					faildAcc.Store(parentAccountId, struct{}{})
					continue
				}

				if subLen := len(subAccLiveCells.Objects); subLen != 1 {
					log.Warn("sub account outpoint len: %d", subLen)
					faildAcc.Store(parentAccountId, struct{}{})
					continue
				}

				subAccountLiveCell := subAccLiveCells.Objects[0]
				if subAccountLiveCell != nil {
					subDataDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
					log.Info("Sync smt server Compare root, parent_id: ", parentAccountId, "t_smt_info root: ", common.Bytes2Hex(currentRoot), " sub_account_cell root: ", common.Bytes2Hex(subDataDetail.SmtRoot))
					if bytes.Compare(currentRoot, subDataDetail.SmtRoot) != 0 {
						faildAcc.Store(parentAccountId, struct{}{})
					}
				}
			}

		}()
	}
	wgTask.Wait()

	faildAcc.Range(func(k, v interface{}) bool {
		resp.SyncFaildAcc = append(resp.SyncFaildAcc, fmt.Sprintf("%s", k))
		return true
	})
	apiResp.ApiRespOK(resp)
	if len(resp.SyncFaildAcc) > 0 {
		return fmt.Errorf("sync faild accountId : %+v", resp.SyncFaildAcc)
	}
	return nil
}
