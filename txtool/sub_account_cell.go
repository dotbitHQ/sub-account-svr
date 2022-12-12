package txtool

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/types"
)

func getSubAccountWitness(subAccountParamList []*witness.SubAccountNew) ([][]byte, error) {
	var witnessList [][]byte
	for _, v := range subAccountParamList {
		wit, _ := v.GenWitness()
		witnessList = append(witnessList, wit)
	}
	return witnessList, nil
}

func (s *SubAccountTxTool) GetOldSubAccount(subAccountIds []string, action common.DasAction) (map[string]string, map[string]*witness.SubAccountNew, error) {
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

	var subAccountBuilderMap = make(map[string]*witness.SubAccountNew)
	var sanb witness.SubAccountNewBuilder
	for _, v := range hashMap {
		res, err := s.DasCore.Client().GetTransaction(s.Ctx, v)
		if err != nil {
			return nil, nil, fmt.Errorf("GetTransaction err: %s", err.Error())
		}
		builderMap, err := sanb.SubAccountNewMapFromTx(res.Transaction) //witness.SubAccountBuilderMapFromTx(res.Transaction)
		if err != nil {
			return nil, nil, fmt.Errorf("SubAccountBuilderMapFromTx err: %s", err.Error())
		}
		for k, bu := range builderMap {
			if item, ok := subAccountBuilderMap[k]; ok {
				if item.CurrentSubAccountData.Nonce < bu.CurrentSubAccountData.Nonce {
					subAccountBuilderMap[k] = builderMap[k]
				}
			} else {
				subAccountBuilderMap[k] = builderMap[k]
			}
		}
	}
	return valueMap, subAccountBuilderMap, nil
}
