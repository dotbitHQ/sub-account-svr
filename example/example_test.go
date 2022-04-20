package example

import (
	"context"
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/sign"
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

func TestTronVerifySignature(t *testing.T) {
	data := common.Hex2Bytes("363637323666366432303634363936343361323034323066333833383230653130376562316635343362383136613134333062333966643063663838666462393865643733386439663134346564383666656661")
	fmt.Println(string(data))

	//0x67180e3b847f126ac8f069c0281122ed2fb668b761f34b9059ec4673d9129cfa185c82973bc6d192a5e2e1f74b3e4df4f78de07d1f669737fb1f061e160b6a081c
	data = common.Hex2Bytes("66726f6d206469643a20420f383820e107eb1f543b816a1430b39fd0cf88fdb98ed738d9f144ed86fefa")
	private := ""
	res, _ := sign.TronSignature(true, data, private)
	fmt.Println("res", common.Bytes2Hex(res))

	//data = common.Hex2Bytes("66726f6d206469643a2066a597e0e651f1249b0d931154b490ee4e5ca69da960acd72bba7d0d19d19b31")
	//signAddress := "TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV"
	//signMsg := common.Hex2Bytes("0x67180e3b847f126ac8f069c0281122ed2fb668b761f34b9059ec4673d9129cfa185c82973bc6d192a5e2e1f74b3e4df4f78de07d1f669737fb1f061e160b6a081c")
	//signMsg = res
	////
	//fmt.Println(sign.TronVerifySignature(true, signMsg, data, signAddress))
}

func TestETH(t *testing.T) {
	signMsg := "0xa6406c20a09e5194e5e20f12fbdaef920051a5d35d24a394077957ab2d6d913e3ca2ef6366439a1aa3ecc6ea41c00b8ffc46c3deb6de8f442b347a625be3958100"
	data := "from did: 09c6905c30db97445c17df9a97079bc5f241c33199aa758e7682f5e9430c8f5c"
	address := "0x15a33588908cf8edb27d1abe3852bf287abd3891"

	fmt.Println(common.Bytes2Hex([]byte(data)))
	fmt.Println(common.Hex2Bytes(data))

	fmt.Println(sign.VerifyPersonalSignature(common.Hex2Bytes(signMsg), common.Hex2Bytes(data), address))
	//data:=common.Hex2Bytes("66726f6d206469643a2030396336393035633330646239373434356331376466396139373037396263356632343163333331393961613735386537363832663565393433306338663563")
	//private:=""
	//fmt.Println(sign.PersonalSignature(data,private))
}

func TestFromDid(t *testing.T) {
	fmt.Println(string(common.Hex2Bytes("66726f6d206469643a20")))
	fmt.Println(common.Bytes2Hex([]byte("from did: ")))
}
