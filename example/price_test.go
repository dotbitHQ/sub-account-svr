package example

import (
	"context"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"sync"
	"testing"
)

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
	return dc, nil
}

func TestGetCustomScriptMintTotalCapacity(t *testing.T) {
	priceApi := txtool.PriceApiDefault{}
	var mintList []tables.TableSmtRecordInfo
	mintList = append(mintList, tables.TableSmtRecordInfo{
		Account:       "12345.a.bit",
		RegisterYears: 1,
	})
	//mintList = append(mintList, tables.TableSmtRecordInfo{
	//	Account:       "1234.a.bit",
	//	RegisterYears: 1,
	//})

	dc, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}
	quoteCell, err := dc.GetQuoteCell()
	if err != nil {
		t.Fatal(err)
	}
	builder, err := dc.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		t.Fatal(err)
	}

	rate, err := molecule.Bytes2GoU32(builder.ConfigCellSubAccount.NewSubAccountCustomPriceDasProfitRate().RawData())
	if err != nil {
		t.Fatal(err)
	}

	res, err := txtool.GetCustomScriptMintTotalCapacity(&txtool.ParamCustomScriptMintTotalCapacity{
		PriceApi:                              &priceApi,
		MintList:                              mintList,
		Quote:                                 quoteCell.Quote(),
		NewSubAccountCustomPriceDasProfitRate: rate,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res.OwnerCapacity, res.DasCapacity)
}

func TestData(t *testing.T) {
	d := witness.ConvertSubAccountCellOutputData(common.Hex2Bytes("0xa17b63dfc8051cbb333936d19a9a9df3e8032f2f8751f53ac249edef81aae94500ea37f006000000000000000000000001f15f519ecb226cd763b2bcbcab093e63f89100c07ac0caebc032c788b187ec99"))
	fmt.Println(d.OwnerProfit, d.DasProfit)
}
