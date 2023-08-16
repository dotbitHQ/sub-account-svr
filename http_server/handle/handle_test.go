package handle

import (
	"context"
	"das_sub_account/config"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/scorpiotzh/toolib"
	"sync"
	"testing"
)

func TestDoSubAccountCheckList(t *testing.T) {
	dc, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}
	var h HttpHandle
	h.DasCore = dc
	config.Cfg.Das.MaxRegisterYears = 20
	req := ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "0001.bit",
		SubAccountList: []CreateSubAccount{
			{
				Account:       "ให้😊บaริก😊02.0001.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
			},
			{
				Account:       "00009.0001.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
			},
		},
	}

	var apiResp api_code.ApiResp
	ok, res, err := h.doSubAccountCheckList(&req, &apiResp)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(res.Result))
	fmt.Println(ok)

}

func getClientTestnet2() (rpc.Client, error) {
	ckbUrl := "https://testnet.ckb.dev/"
	indexerUrl := "https://testnet.ckb.dev/indexer"
	return rpc.DialWithIndexer(ckbUrl, indexerUrl)
}

func getNewDasCoreTestnet2() (*core.DasCore, error) {
	client, err := getClientTestnet2()
	if err != nil {
		return nil, err
	}

	env := core.InitEnvOpt(common.DasNetTypeTestnet2,
		common.DasContractNameConfigCellType,
	)
	var wg sync.WaitGroup
	ops := []core.DasCoreOption{
		core.WithClient(client),
		core.WithDasContractArgs(env.ContractArgs),
		core.WithDasContractCodeHash(env.ContractCodeHash),
		core.WithDasNetType(common.DasNetTypeTestnet2),
		core.WithTHQCodeHash(env.THQCodeHash),
	}
	dc := core.NewDasCore(context.Background(), &wg, ops...)
	// contract
	dc.InitDasContract(env.MapContract)
	// config cell
	if err = dc.InitDasConfigCell(); err != nil {
		return nil, err
	}
	// so script
	if err = dc.InitDasSoScript(); err != nil {
		return nil, err
	}
	return dc, nil
}
