package txtool

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
)

func getSubAccountWitness(subAccountParamList []*witness.SubAccountParam) ([][]byte, error) {
	var witnessList [][]byte
	for _, v := range subAccountParamList {
		log.Info("getSubAccountWitness:", toolib.JsonString(v))
		wit, _ := v.NewSubAccountWitness()
		witnessList = append(witnessList, wit)
	}
	return witnessList, nil
}

func (s *SubAccountTxTool) GetOldSubAccount(subAccountIds []string, action common.DasAction) (map[string]string, map[string]*witness.SubAccountBuilder, error) {
	smtInfoList, err := s.DbDao.GetSmtInfoBySubAccountIds(subAccountIds)
	if err != nil {
		return nil, nil, fmt.Errorf("GetSmtInfoBySubAccountIds err: %s", err.Error())
	}
	var valueMap = make(map[string]string)
	var hashMap = make(map[string]types.Hash)
	for _, v := range smtInfoList {
		res := common.String2OutPointStruct(v.Outpoint)
		hashMap[res.TxHash.Hex()] = res.TxHash
		valueMap[v.AccountId] = v.LeafDataHash
	}

	var subAccountBuilderMap = make(map[string]*witness.SubAccountBuilder)
	if action == common.DasActionEditSubAccount {
		for _, v := range hashMap {
			res, err := s.DasCore.Client().GetTransaction(s.Ctx, v)
			if err != nil {
				return nil, nil, fmt.Errorf("GetTransaction err: %s", err.Error())
			}
			builderMap, err := witness.SubAccountBuilderMapFromTx(res.Transaction)
			if err != nil {
				return nil, nil, fmt.Errorf("SubAccountBuilderMapFromTx err: %s", err.Error())
			}
			for k, _ := range builderMap {
				subAccountBuilderMap[k] = builderMap[k]
			}
		}
	}
	return valueMap, subAccountBuilderMap, nil
}
