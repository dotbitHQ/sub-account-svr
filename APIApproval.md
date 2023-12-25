* [API for Approval](#api-for-approval)
  * [Approval Enable](#Approval-Enable)
  * [Approval Delay](#Approval-Delay)
  * [Approval Revoke](#Approval-Revoke)
  * [Approval Fulfill](#Approval-Fulfill)
* [API for Transaction](#api-for-transaction)
  * [Send Transaction](#send-transaction)
  * [Transaction Status](#transaction-status)

## Instructions For Use
1. call [API for Approval](#api-for-approval) get data waiting for signature
2. use [signTxList](https://github.com/dotbitHQ/wallet-bridge?tab=readme-ov-file#46-initsigncontext-promiseinitsigncontextres) get sign result
3. call [Send Transaction](#send-transaction) to send transaction
4. call [Transaction Status](#transaction-status) get transaction status

## API-for-Approval
    
### Approval-Enable

#### Request

* path: /v1/approval/enable

Params:
- platform: platform key info
    - type: blockchain
        - key_info
            - coin_type   `platform coin_type only can be '60'`
            - key
- owner: `account owner key info`
- to: `account to key info`
- account: `account name`
- protected_until: `protected until time, authorization irrevocable time, before this time can not call` [Approval Revoke](#Approval-Revoke)
- sealed_until: `sealed until time, authorization effective time, after this time every one can call` [Approval Fulfill](#Approval-Fulfill)
- evm_chain_id: `evm chain id, only the main account need this parameter`
```json
{
  "platform": {
    "type": "blockchain",
    "key_info": {
      "coin_type": "60",
      "key": "0xe58673b9bF0a57398e0C8A1BDAe01EEB730177C8"
    }
  },
  "owner": {
    "type": "blockchain",
    "key_info": {
      "coin_type": "60",
      "key": "0xdeeFC10a42cD84c072f2b0e2fA99061a74A0698c"
    }
  },
  "to": {
    "type": "blockchain",
    "key_info": {
      "coin_type": "60",
      "key": "0x52045950a5B582E9b426Ad89296c8970c96D09D9"
    }
  },
  "account": "cross15.bit",
  "protected_until": 1692696089,
  "sealed_until": 1692762911,
  "evm_chain_id": 5
}
```

#### Response

##### main_account
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "create_approval",
    "sub_action": "",
    "sign_key": "c5d332a1cf42cf066c49849cc91f83d6",
    "sign_address": "",
    "is_712": true,
    "sign_list": [
      {
        "sign_type": 5,
        "sign_msg": "0x63c1d729b4e293ecef164dabceb1ab3e2be62f5117a036ab1c37d4eb1698ff85"
      }
    ],
    "mm_json": {
      "types": {
        "EIP712Domain": [
          {
            "name": "chainId",
            "type": "uint256"
          },
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "verifyingContract",
            "type": "address"
          },
          {
            "name": "version",
            "type": "string"
          }
        ],
        "Action": [
          {
            "name": "action",
            "type": "string"
          },
          {
            "name": "params",
            "type": "string"
          }
        ],
        "Cell": [
          {
            "name": "capacity",
            "type": "string"
          },
          {
            "name": "lock",
            "type": "string"
          },
          {
            "name": "type",
            "type": "string"
          },
          {
            "name": "data",
            "type": "string"
          },
          {
            "name": "extraData",
            "type": "string"
          }
        ],
        "Transaction": [
          {
            "name": "DAS_MESSAGE",
            "type": "string"
          },
          {
            "name": "inputsCapacity",
            "type": "string"
          },
          {
            "name": "outputsCapacity",
            "type": "string"
          },
          {
            "name": "fee",
            "type": "string"
          },
          {
            "name": "action",
            "type": "Action"
          },
          {
            "name": "inputs",
            "type": "Cell[]"
          },
          {
            "name": "outputs",
            "type": "Cell[]"
          },
          {
            "name": "digest",
            "type": "bytes32"
          }
        ]
      },
      "primaryType": "Transaction",
      "domain": {
        "chainId": 5,
        "name": "da.systems",
        "verifyingContract": "0x0000000000000000000000000000000020210722",
        "version": "1"
      },
      "message": {
        "DAS_MESSAGE": "APPROVE TRANSFER sub-account-test.bit TO 0x52045950a5b582e9b426ad89296c8970c96d09d9 AFTER 1692870747",
        "inputsCapacity": "226.99988747 CKB",
        "outputsCapacity": "226.99983611 CKB",
        "fee": "0.00005136 CKB",
        "digest": "",
        "action": {
          "action": "create_approval",
          "params": "0x00"
        },
        "inputs": [
          {
            "capacity": "226.99988747 CKB",
            "lock": "das-lock,0x01,0x05deefc10a42cd84c072f2b0e2fa99061a74a069...",
            "type": "account-cell-type,0x01,0x",
            "data": "{ account: sub-account-test.bit, expired_at: 2028621581 }",
            "extraData": "{ status: 0, records_hash: 0x0a5e0d314f2871334d8e3f5d49b2af60c49ac9af594debc705522448c5722ebf }"
          }
        ],
        "outputs": [
          {
            "capacity": "226.99983611 CKB",
            "lock": "das-lock,0x01,0x05deefc10a42cd84c072f2b0e2fa99061a74a069...",
            "type": "account-cell-type,0x01,0x",
            "data": "{ account: sub-account-test.bit, expired_at: 2028621581 }",
            "extraData": "{ status: 4, records_hash: 0x0a5e0d314f2871334d8e3f5d49b2af60c49ac9af594debc705522448c5722ebf }"
          }
        ]
      }
    }
  }
}
```

replace `sign_list[0].sign_msg` to `mm_json.message.digest` and sign mm_json content, like this
```json
{
      "types": {
        "EIP712Domain": [
          {
            "name": "chainId",
            "type": "uint256"
          },
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "verifyingContract",
            "type": "address"
          },
          {
            "name": "version",
            "type": "string"
          }
        ],
        "Action": [
          {
            "name": "action",
            "type": "string"
          },
          {
            "name": "params",
            "type": "string"
          }
        ],
        "Cell": [
          {
            "name": "capacity",
            "type": "string"
          },
          {
            "name": "lock",
            "type": "string"
          },
          {
            "name": "type",
            "type": "string"
          },
          {
            "name": "data",
            "type": "string"
          },
          {
            "name": "extraData",
            "type": "string"
          }
        ],
        "Transaction": [
          {
            "name": "DAS_MESSAGE",
            "type": "string"
          },
          {
            "name": "inputsCapacity",
            "type": "string"
          },
          {
            "name": "outputsCapacity",
            "type": "string"
          },
          {
            "name": "fee",
            "type": "string"
          },
          {
            "name": "action",
            "type": "Action"
          },
          {
            "name": "inputs",
            "type": "Cell[]"
          },
          {
            "name": "outputs",
            "type": "Cell[]"
          },
          {
            "name": "digest",
            "type": "bytes32"
          }
        ]
      },
      "primaryType": "Transaction",
      "domain": {
        "chainId": 5,
        "name": "da.systems",
        "verifyingContract": "0x0000000000000000000000000000000020210722",
        "version": "1"
      },
      "message": {
        "DAS_MESSAGE": "APPROVE TRANSFER sub-account-test.bit TO 0x52045950a5b582e9b426ad89296c8970c96d09d9 AFTER 1692870747",
        "inputsCapacity": "226.99988747 CKB",
        "outputsCapacity": "226.99983611 CKB",
        "fee": "0.00005136 CKB",
        "digest": "0x63c1d729b4e293ecef164dabceb1ab3e2be62f5117a036ab1c37d4eb1698ff85",
        "action": {
          "action": "create_approval",
          "params": "0x00"
        },
        "inputs": [
          {
            "capacity": "226.99988747 CKB",
            "lock": "das-lock,0x01,0x05deefc10a42cd84c072f2b0e2fa99061a74a069...",
            "type": "account-cell-type,0x01,0x",
            "data": "{ account: sub-account-test.bit, expired_at: 2028621581 }",
            "extraData": "{ status: 0, records_hash: 0x0a5e0d314f2871334d8e3f5d49b2af60c49ac9af594debc705522448c5722ebf }"
          }
        ],
        "outputs": [
          {
            "capacity": "226.99983611 CKB",
            "lock": "das-lock,0x01,0x05deefc10a42cd84c072f2b0e2fa99061a74a069...",
            "type": "account-cell-type,0x01,0x",
            "data": "{ account: sub-account-test.bit, expired_at: 2028621581 }",
            "extraData": "{ status: 4, records_hash: 0x0a5e0d314f2871334d8e3f5d49b2af60c49ac9af594debc705522448c5722ebf }"
          }
        ]
      }
    }
```
and call [Send Transaction](#send-transaction)

##### sub_account
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "update_sub_account",
    "sub_action": "create_approval",
    "sign_key": "c5d332a1cf42cf066c49849cc91f83d6",
    "sign_address": "",
    "is_712": false,
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "From .bit: 0x63c1d729b4e293ecef164dabceb1ab3e2be62f5117a036ab1c37d4eb1698ff85"
      }
    ]
  }
}
```
personal sign with `sign_list[0].sign_msg` and call [Send Transaction](#send-transaction)

### Approval-Delay

#### Request

* path: /v1/approval/delay

```json
{
  "type": "blockchain",
  "key_info": {         // owner key_info
    "coin_type": "60",
    "key": "0xdeeFC10a42cD84c072f2b0e2fA99061a74A0698c"
  },
  "account": "cross15.bit",
  "sealed_until": 1692762911, // extend approval effective time
  "evm_chain_id": 5    // only the main account need this parameter
}
```

#### Response

##### main_account
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "delay_approval",
    "sub_action": "",
    "sign_key": "e6358f9798aea2182329657029e6ff84",
    "sign_address": "",
    "is_712": true,
    "sign_list": [
      {
        "sign_type": 5,
        "sign_msg": "0xa67ee3e2a14602ca7dfa8d720c5c59c0769c8535575b4832aa46bd8f0023b476"
      }
    ],
    "mm_json": {
      "types": {
        "EIP712Domain": [
          {
            "name": "chainId",
            "type": "uint256"
          },
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "verifyingContract",
            "type": "address"
          },
          {
            "name": "version",
            "type": "string"
          }
        ],
        "Action": [
          {
            "name": "action",
            "type": "string"
          },
          {
            "name": "params",
            "type": "string"
          }
        ],
        "Cell": [
          {
            "name": "capacity",
            "type": "string"
          },
          {
            "name": "lock",
            "type": "string"
          },
          {
            "name": "type",
            "type": "string"
          },
          {
            "name": "data",
            "type": "string"
          },
          {
            "name": "extraData",
            "type": "string"
          }
        ],
        "Transaction": [
          {
            "name": "DAS_MESSAGE",
            "type": "string"
          },
          {
            "name": "inputsCapacity",
            "type": "string"
          },
          {
            "name": "outputsCapacity",
            "type": "string"
          },
          {
            "name": "fee",
            "type": "string"
          },
          {
            "name": "action",
            "type": "Action"
          },
          {
            "name": "inputs",
            "type": "Cell[]"
          },
          {
            "name": "outputs",
            "type": "Cell[]"
          },
          {
            "name": "digest",
            "type": "bytes32"
          }
        ]
      },
      "primaryType": "Transaction",
      "domain": {
        "chainId": 5,
        "name": "da.systems",
        "verifyingContract": "0x0000000000000000000000000000000020210722",
        "version": "1"
      },
      "message": {
        "DAS_MESSAGE": "DELAY THE TRANSFER APPROVAL OF sub-account-test.bit TO 1693022303",
        "inputsCapacity": "226.99983611 CKB",
        "outputsCapacity": "226.99978213 CKB",
        "fee": "0.00005398 CKB",
        "digest": "",
        "action": {
          "action": "delay_approval",
          "params": "0x00"
        },
        "inputs": [
          {
            "capacity": "226.99983611 CKB",
            "lock": "das-lock,0x01,0x05deefc10a42cd84c072f2b0e2fa99061a74a069...",
            "type": "account-cell-type,0x01,0x",
            "data": "{ account: sub-account-test.bit, expired_at: 2028621581 }",
            "extraData": "{ status: 4, records_hash: 0x0a5e0d314f2871334d8e3f5d49b2af60c49ac9af594debc705522448c5722ebf }"
          }
        ],
        "outputs": [
          {
            "capacity": "226.99978213 CKB",
            "lock": "das-lock,0x01,0x05deefc10a42cd84c072f2b0e2fa99061a74a069...",
            "type": "account-cell-type,0x01,0x",
            "data": "{ account: sub-account-test.bit, expired_at: 2028621581 }",
            "extraData": "{ status: 4, records_hash: 0x0a5e0d314f2871334d8e3f5d49b2af60c49ac9af594debc705522448c5722ebf }"
          }
        ]
      }
    }
  }
}
```

##### sub_account
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "update_sub_account",
    "sub_action": "delay_approval",
    "sign_key": "c5d332a1cf42cf066c49849cc91f83d6",
    "sign_address": "",
    "is_712": false,
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "From .bit: 0x63c1d729b4e293ecef164dabceb1ab3e2be62f5117a036ab1c37d4eb1698ff85"
      }
    ]
  }
}
```

### Approval-Revoke

#### Request

* path: /v1/approval/revoke

```json
{
  "type": "blockchain",
  "key_info": {         // platform key_info
    "coin_type": "60",  // coin_type only can be '60'
    "key": "0xdeeFC10a42cD84c072f2b0e2fA99061a74A0698c"
  },
  "account": "cross15.bit"
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "revoke_approval",
    "sub_action": "",
    "sign_key": "5770c64eb8d45fe84e1578908694db14",
    "sign_address": "",
    "is_712": false,
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "From .bit: 2ac9958d0f3230869b8724be9ba0b87e84bfc6aaa2e8ee06f5f43ab9c0ea0593"
      }
    ]
  }
}
```

### Approval-Fulfill

#### Request

* path: /v1/approval/fulfill

```json
{
  "type": "blockchain",
  "key_info": {         // owner key_info
    "coin_type": "60",
    "key": "0xdeeFC10a42cD84c072f2b0e2fA99061a74A0698c"
  },
  "account": "cross15.bit",
  "evm_chain_id": 5    // only the main account need this parameter
}
```

#### Response

##### main_account
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "fulfill_approval",
    "sub_action": "",
    "sign_key": "35589b94e1571c946792c595b74e84ab",
    "sign_address": "",
    "is_712": true,
    "sign_list": [
      {
        "sign_type": 5,
        "sign_msg": "0x35ad242cc7e2ceb4e07aaa7129473b0ad0b10ab7579e3cd66ae0d313ff97c591"
      }
    ],
    "mm_json": {
      "types": {
        "EIP712Domain": [
          {
            "name": "chainId",
            "type": "uint256"
          },
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "verifyingContract",
            "type": "address"
          },
          {
            "name": "version",
            "type": "string"
          }
        ],
        "Action": [
          {
            "name": "action",
            "type": "string"
          },
          {
            "name": "params",
            "type": "string"
          }
        ],
        "Cell": [
          {
            "name": "capacity",
            "type": "string"
          },
          {
            "name": "lock",
            "type": "string"
          },
          {
            "name": "type",
            "type": "string"
          },
          {
            "name": "data",
            "type": "string"
          },
          {
            "name": "extraData",
            "type": "string"
          }
        ],
        "Transaction": [
          {
            "name": "DAS_MESSAGE",
            "type": "string"
          },
          {
            "name": "inputsCapacity",
            "type": "string"
          },
          {
            "name": "outputsCapacity",
            "type": "string"
          },
          {
            "name": "fee",
            "type": "string"
          },
          {
            "name": "action",
            "type": "Action"
          },
          {
            "name": "inputs",
            "type": "Cell[]"
          },
          {
            "name": "outputs",
            "type": "Cell[]"
          },
          {
            "name": "digest",
            "type": "bytes32"
          }
        ]
      },
      "primaryType": "Transaction",
      "domain": {
        "chainId": 5,
        "name": "da.systems",
        "verifyingContract": "0x0000000000000000000000000000000020210722",
        "version": "1"
      },
      "message": {
        "DAS_MESSAGE": "FULFILL THE TRANSFER APPROVAL OF sub-account-test.bit, TRANSFER TO 0x52045950a5b582e9b426ad89296c8970c96d09d9",
        "inputsCapacity": "226.99983611 CKB",
        "outputsCapacity": "226.99978658 CKB",
        "fee": "0.00004953 CKB",
        "digest": "",
        "action": {
          "action": "fulfill_approval",
          "params": "0x00"
        },
        "inputs": [
          {
            "capacity": "226.99983611 CKB",
            "lock": "das-lock,0x01,0x05deefc10a42cd84c072f2b0e2fa99061a74a069...",
            "type": "account-cell-type,0x01,0x",
            "data": "{ account: sub-account-test.bit, expired_at: 2028621581 }",
            "extraData": "{ status: 4, records_hash: 0x0a5e0d314f2871334d8e3f5d49b2af60c49ac9af594debc705522448c5722ebf }"
          }
        ],
        "outputs": [
          {
            "capacity": "226.99978658 CKB",
            "lock": "das-lock,0x01,0x0552045950a5b582e9b426ad89296c8970c96d09...",
            "type": "account-cell-type,0x01,0x",
            "data": "{ account: sub-account-test.bit, expired_at: 2028621581 }",
            "extraData": "{ status: 0, records_hash: 0x55478d76900611eb079b22088081124ed6c8bae21a05dd1a0d197efcc7c114ce }"
          }
        ]
      }
    }
  }
}
```

##### sub_account
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "update_sub_account",
    "sub_action": "fulfill_approval",
    "sign_key": "c5d332a1cf42cf066c49849cc91f83d6",
    "sign_address": "",
    "is_712": false,
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "From .bit: 0x63c1d729b4e293ecef164dabceb1ab3e2be62f5117a036ab1c37d4eb1698ff85"
      }
    ]
  }
}
```

## API for Transaction

### Send Transaction

#### Request

* path: /v1/transaction/send
```json
{
  "action": "create_approval",    // same with the api return
  "sub_action": "",               // same with the api return
  "sign_address": "0x111...",     // only sign_type=8 webauthn, You need to fill in this address
  "sign_key": "18feccf2347ed980f07bd3277f9ce626", // same with the api return
  "sign_list": [
    {
      "sign_type": 5,  // same with the api return sign_list[0].sign_type
      "sign_msg": "0x0ea5ffd13bddbdb3f5a8b492cd6653816d371b9afebb7e6d4ecd8e2962d6b4ca" // signature result
    }
  ]
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "hash_list": [""]
  }
}
```

### Transaction Status

#### Request

* path: /v1/transaction/status

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0x111..."
  },
  "action": "enable_sub_account",
  "sub_action": "",
  "account": ""
}
```

#### Response

is pending or unsend

* status: 0: pending, 2: unsend
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "block_number": 0,
    "hash": "",
    "status": 0
  }
}
```

is committed
```json
{
  "err_no": 11001,
  "err_msg": "not exits tx",
  "data": null
}
```
