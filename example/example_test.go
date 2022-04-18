package example

import (
	"context"
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/smt"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/scorpiotzh/toolib"
	"sort"
	"strings"
	"testing"
)

func getClientTestnet2() (rpc.Client, error) {
	ckbUrl := "http://47.243.90.165:8114"
	indexerUrl := "http://47.243.90.165:8116"
	return rpc.DialWithIndexer(ckbUrl, indexerUrl)
}

func TestAccountId(t *testing.T) {
	accountId := "0x16fcb57af932d4b5729224627105cc1165a9bf90"
	key := smt.AccountIdToSmtH256(accountId)
	fmt.Println(len(key), key, common.Bytes2Hex(key))
}

func TestGetLiveCell(t *testing.T) {
	client, err := getClientTestnet2()
	if err != nil {
		t.Fatal(err)
	}
	outpoint := common.String2OutPointStruct("0x1ea8c5e9dfbd40a4843c60894abcff7797dfd95e8190b508d02a6f633d306b91-0")
	res, err := client.GetLiveCell(context.Background(), outpoint, false)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(res))
	//{"cell":{"data":null,"output":{"capacity":10600000000,"lock":{"code_hash":"0xf1ef61b6977508d9ec56fe43399a01e576086a76cf0f7c687d1418335e8c401f","hash_type":"type","args":""},"type":{"code_hash":"0x67d48c0911e406518de2116bd91c6af37c05f1db23334ca829d2af3042427e44","hash_type":"type","args":""}}},"status":"live"}
}

func TestAccount(t *testing.T) {
	account := "00001.0001.bit"
	fmt.Println(account[strings.Index(account, "."):])
	accountCharSet, err := common.AccountToAccountChars(account[:strings.Index(account, ".")])
	fmt.Println(toolib.JsonString(accountCharSet), err)
}

func TestTask(t *testing.T) {
	task := tables.TableTaskInfo{
		Id:              0,
		TaskId:          "",
		TaskType:        tables.TaskTypeChain,
		ParentAccountId: "0x338e9410a195ddf7fedccd99834ea6c5b6e5449c",
		Action:          common.DasActionEnableSubAccount,
		RefOutpoint:     "",
		BlockNumber:     0,
		Outpoint:        "0xd2ed490f6cec9543291b3b730d0f38a2e46258c8848c6ec7ac12a6f9fa0ffd7f-1",
		Timestamp:       1648102865237,
		SmtStatus:       tables.SmtStatusWriteComplete,
		TxStatus:        tables.TxStatusCommitted,
	}
	task.InitTaskId()
	fmt.Println(task.TaskId)
}

func TestEditRecord(t *testing.T) {
	list := []handle.EditRecord{
		{
			Index: 1,
			Key:   "aaaa",
			Type:  "",
			Label: "",
			Value: "",
			TTL:   "",
		},
		{
			Index: 0,
			Key:   "bbbb",
			Type:  "",
			Label: "",
			Value: "",
			TTL:   "",
		},
	}
	fmt.Println(list)
	sort.Sort(handle.EditRecordList(list))
	fmt.Println(list)
}
