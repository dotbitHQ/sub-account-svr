package example

import (
	"context"
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

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
	_, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}

	account := "0😊😊0😊0😊0😊1😊.0001.bit"
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

func TestAccountLen(t *testing.T) {
	db, err := toolib.NewGormDB("", "", "", "das_database", 100, 200)
	if err != nil {
		t.Fatal(err)
	}
	var list []tables.TableAccountInfo
	err = db.Where("parent_account_id=''").Order("registered_at").Find(&list).Error
	if err != nil {
		t.Fatal(err)
	}
	type RegisterInfo struct {
		Count4     uint64
		Count5     uint64
		CountOwner uint64
	}

	//fmt.Println("account list:", len(list))
	var res = make(map[string]RegisterInfo)
	var owner = make(map[string]struct{})
	for _, v := range list {
		_, length, _ := common.GetDotBitAccountLength(v.Account)
		tm := time.Unix(int64(v.RegisteredAt), 0)
		registeredAt := tm.Format("2006-01-02")
		var tmp RegisterInfo
		if item, ok := res[registeredAt]; ok {
			tmp.Count4 = item.Count4
			tmp.Count5 = item.Count5
			tmp.CountOwner = item.CountOwner
		}
		if length == 4 {
			tmp.Count4++
		} else if length >= 5 {
			tmp.Count5++
		}

		if _, ok := owner[strings.ToLower(v.Owner)]; !ok {
			tmp.CountOwner++
			owner[strings.ToLower(v.Owner)] = struct{}{}
		}
		res[registeredAt] = tmp
	}

	count := uint64(0)
	var strList []string

	for k, v := range res {
		strList = append(strList, fmt.Sprintf("%s,%d,%d,%d,\n", k, v.Count4, v.Count5, v.CountOwner))
		count += v.CountOwner
	}
	fmt.Println("count:", count)
	sort.Strings(strList)
	txtPath := "./register-info.csv"
	for _, v := range strList {
		//fmt.Println(v)
		if err := writeToFile(txtPath, v); err != nil {
			fmt.Println(err)
		}
	}
}

func TestAccountLen2(t *testing.T) {
	db, err := toolib.NewGormDB("", "", "", "das_database", 100, 200)
	if err != nil {
		t.Fatal(err)
	}
	var list []tables.TableAccountInfo
	err = db.Where("parent_account_id=''").Order("registered_at").Find(&list).Error
	if err != nil {
		t.Fatal(err)
	}
	txtPath := "./register.csv"
	for _, v := range list {
		timeAt := (v.ExpiredAt - v.RegisteredAt) / uint64(common.OneYearSec)
		_, length, _ := common.GetDotBitAccountLength(v.Account)
		lengthStr := "4位"
		if length > 4 {
			lengthStr = "5位及以上"
		}
		tm := time.Unix(int64(v.RegisteredAt), 0)
		registeredAt := tm.Format("2006-01-02")
		msg := fmt.Sprintf("%d,%s,%s,%s,%d,\n", v.Id, v.Account, lengthStr, registeredAt, timeAt)
		if err := writeToFile(txtPath, msg); err != nil {
			fmt.Println(err)
		}
	}
}

func writeToFile(fileName, msg string) error {
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("os.OpenFile err: %s", err.Error())
	} else {
		defer f.Close()
		n, _ := f.Seek(0, io.SeekEnd)
		_, err = f.WriteAt([]byte(msg), n)
		if err != nil {
			return fmt.Errorf("f.WriteAt err: %s", err.Error())
		}
	}
	return nil
}

func TestAccountIndex(t *testing.T) {
	acc := "aaaaa.bit"
	index := strings.Index(acc, ".")
	fmt.Println(index)
	fmt.Println(acc[index:])
	suffix := strings.TrimLeft(acc[index:], ".")

	fmt.Println(suffix, acc[strings.Index(acc, ".")+1:])
}

func TestDasLockBalance(t *testing.T) {
	db, err := toolib.NewGormDB("", "", "", "das_database", 100, 200)
	if err != nil {
		t.Fatal(err)
	}
	var list []tables.TableAccountInfo
	err = db.Select("owner_chain_type,owner").Group("owner_chain_type,owner").Find(&list).Error
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(list))
	//
	dc, err := getNewDasCoreMainnet()
	if err != nil {
		t.Fatal(err)
	}
	//
	ch := make(chan tables.TableAccountInfo, 100)

	total := uint64(0)
	errGroup := &errgroup.Group{}
	lock := &sync.Mutex{}
	for i := 0; i < 30; i++ {
		fmt.Println(i)
		errGroup.Go(func() error {
			for acc := range ch {
				if acc.Owner == common.BlackHoleAddress {
					continue
				}
				chainType := acc.OwnerChainType
				owner := acc.Owner
				//fmt.Println(chainType, owner)
				balance, err := getBalance(dc, chainType, owner)
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				if balance > 0 {
					lock.Lock()
					total += balance
					lock.Unlock()
				}
				fmt.Println(acc.Owner, balance)
			}
			return nil
		})
	}

	for i := range list {
		ch <- list[i]
	}
	close(ch)

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	fmt.Println("total:", total)

	//fmt.Println(getBalance(dc, common.ChainTypeEth, "0x9176aCD39A3A9Ae99dcB3922757f8Af4f94cDF3C"))
	//fmt.Println(getBalance05(dc))
}

func TestDasLockBalance2(t *testing.T) {
	dc, err := getNewDasCoreMainnet()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(getBalance(dc, common.ChainTypeEth, ""))
	fmt.Println(getBalance05(dc))
}

func getBalance(dc *core.DasCore, chainType common.ChainType, addr string) (uint64, error) {
	total := uint64(0)
	dasLock, _, err := dc.Daf().HexToScript(core.DasAddressHex{
		DasAlgorithmId: chainType.ToDasAlgorithmId(false),
		AddressHex:     addr,
		IsMulti:        false,
		ChainType:      chainType,
	})
	if err != nil {
		return 0, fmt.Errorf("hexToScript err: %s", err.Error())
	}
	searchKey := &indexer.SearchKey{
		Script:     dasLock,
		ScriptType: indexer.ScriptTypeLock,
		Filter: &indexer.CellsFilter{
			OutputDataLenRange: &[2]uint64{0, 1},
		},
	}
	res, err := dc.Client().GetCellsCapacity(context.Background(), searchKey)
	if err != nil {
		return 0, fmt.Errorf("GetCellsCapacity err: %s", err.Error())
	}
	total += res.Capacity
	return total / 1e8, nil
}

func getBalance05(dc *core.DasCore) (uint64, error) {
	total := uint64(0)
	searchKey := &indexer.SearchKey{
		Script: &types.Script{
			CodeHash: types.HexToHash("0xebafc1ebe95b88cac426f984ed5fce998089ecad0cd2f8b17755c9de4cb02162"),
			HashType: types.HashTypeType,
			Args:     nil,
		},
		ScriptType: indexer.ScriptTypeType,
	}
	res, err := dc.Client().GetCellsCapacity(context.Background(), searchKey)
	if err != nil {
		return 0, fmt.Errorf("GetCellsCapacity err: %s", err.Error())
	}
	total += res.Capacity
	return total / 1e8, nil
}
