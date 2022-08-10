* [API LIST](#api-list)
    * [Get Config Info](#get-config-info)
    * [Get Account List](#get-account-list)
    * [Get Account Detail](#get-account-detail)
    * [Get Sub Account List](#get-sub-account-list)
    * [Init Sub Account](#init-sub-account)
    * [Check Sub Account](#check-sub-account)
    * [Create Sub Account](#create-sub-account)
    * [Edit Sub Account](#edit-sub-account)
    * [Send Transaction](#send-transaction)
    * [Transaction Status](#transaction-status)
    * [Task Status](#task-status)
    * [Sub Account Mint Status](#sub-account-mint-status)
    * [Custom Script Set](#custom-script-set)
    * [Custom Script Info](#custom-script-info)
    * [Custom Script Price](#custom-script-price)
    * [Owner Profit](#owner-profit)
    * [Profit Withdraw](#profit-withdraw)
* [INTERNAL API LIST](#internal-api-list)
    * [Internal Mint Sub Account](#internal-mint-sub-account)
    * [Internal Check Smt Info](#internal-check-smt-info)
    * [Internal Update Smt](#internal-update-smt)

## API LIST

Please familiarize yourself with the meaning of some common parameters before reading the API list:

| param                                                                                    | description                                                         |
| :-------------------------                                                               | :------------------------------------------------------------------ |
| type                                                                                     | Filled with "blockchain" for now                                    |
| coin\_type <sup>[1](https://github.com/satoshilabs/slips/blob/master/slip-0044.md)</sup> | 60: eth, 195: trx, 9006: bsc, 966: matic                             |
| chain\_id <sup>[2](https://github.com/ethereum-lists/chains)</sup>                       | 1: eth, 56: bsc, 137: polygon; 5: goerli, 97: bsct, 80001: mumbai   |
| account                                                                                  | Contains the suffix `.bit` in it                                    |
| key                                                                                      | Generally refers to the blockchain address for now                  |

_You can provide either `coin_type` or `chain_id`. The `coin_type` will be used, if you provide both_

### Get Config Info

#### Request

* path: /v1/config/info
* params: null

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "sub_account_basic_capacity": 0,
    "sub_account_prepared_fee_capacity": 0,
    "sub_account_new_sub_account_price": 0,
    "sub_account_renew_sub_account_price": 0,
    "sub_account_common_fee": 0
  }
}
```

### Get Account List

#### Request

* path: /v1/account/list
    * category: 1-main account 2-sub account 6-enable sub account

```json
{
  "page": 1,
  "size": 100,
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111...",
    "category": 6
  }
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "list": [
      {
        "account": "",
        "owner": {
          "type": "blockchain",
          "key_info": {
            "coin_type": "60",
            "chain_id": "1",
            "key": "0x111..."
          }
        },
        "manager": {
          "type": "blockchain",
          "key_info": {
            "coin_type": "60",
            "chain_id": "1",
            "key": "0x111..."
          }
        },
        "registered_at": 0,
        "expired_at": 0,
        "status": 0,
        "enable_sub_account": 0,
        "renew_sub_account_price": 0,
        "nonce": 0
      }
    ]
  }
}
```

### Get Account Detail

#### Request

* path: /v1/account/detail

```json
{
  "account": ""
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "account_info": {
      "account": "",
      "owner": {
        "type": "blockchain",
        "key_info": {
          "coin_type": "60",
          "chain_id": "1",
          "key": "0x111..."
        }
      },
      "manager": {
        "type": "blockchain",
        "key_info": {
          "coin_type": "60",
          "chain_id": "1",
          "key": "0x111..."
        }
      },
      "registered_at": 0,
      "expired_at": 0,
      "status": 0,
      "enable_sub_account": 0,
      "renew_sub_account_price": 0,
      "nonce": 0
    },
    "records": [
      {
        "key": "",
        "type": "",
        "label": "",
        "value": "",
        "ttl": ""
      }
    ]
  }
}
```

### Get Sub Account List

#### Request

* path: /v1/sub/account/list
* params:
    * key_info: not necessary
    * type: not necessary
    * CategoryExpireSoon=4
    * CategoryToBeRecycled=5
    * order_type:
        * OrderTypeAccountAsc OrderType = 0
        * OrderTypeAccountDesc OrderType = 1
        * OrderTypeRegisterAtAsc OrderType = 2
        * OrderTypeRegisterAtDesc OrderType = 3
        * OrderTypeExpiredAtAsc OrderType = 4
        * OrderTypeExpiredAtDesc OrderType = 5

```json
{
  "page": 1,
  "size": 100,
  "account": "",
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111..."
  },
  "keyword": "",
  "category": 0,
  "order_type": 0
}
```

* Return all sub-account of `account`, which belong to `key_info`, if provide `key_info` and `type`
* Return all sub-account of `account`, if not provide `key_info` and `type`

#### Response

* status：0-normal, 1-on sale, 2-on auction, 3-cross opensea

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "list": [
      {
        "account": "",
        "owner": {
          "type": "blockchain",
          "key_info": {
            "coin_type": "60",
            "chain_id": "1",
            "key": "0x111..."
          }
        },
        "manager": {
          "type": "blockchain",
          "key_info": {
            "coin_type": "60",
            "chain_id": "1",
            "key": "0x111..."
          }
        },
        "registered_at": 0,
        "expired_at": 0,
        "status": 0,
        "enable_sub_account": 0,
        "renew_sub_account_price": 0,
        "nonce": 0
      }
    ]
  }
}
```

### Init Sub Account

#### Request

* path: /v1/sub/account/init

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111..."
  },
  "account": ""
}
```

#### Response

* parameters `action` and `sign_key` are required when send transactions

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "action": "enable_sub_account",
    "sign_key": "",
    "list": [
      {
        "sign_list": [
          {
            "sign_type": 3,
            "sign_msg": "from did: 0x123"
          }
        ]
      }
    ]
  }
}
```

### Check Sub Account

#### Request

* path: /v1/sub/account/check

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111..."
  },
  "account": "",
  "sub_account_list": [
    {
      "account": "",
      "mint_for_account": "",
      "account_char_str": [
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        }
      ],
      "register_years": 1,
      "type": "blockchain",
      "key_info": {
        "coin_type": "60",
        "chain_id": "1",
        "key": "0x111..."
      }
    }
  ]
}
```

#### Response

* status: 0: ok, 1: fail, 2: registered, 3: registering

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "result": [
      {
        "account": "",
        "register_years": 1,
        "type": "blockchain",
        "key_info": {
          "coin_type": "60",
          "chain_id": "1",
          "key": "0x111..."
        },
        "status": 1,
        "message": ""
      }
    ]
  }
}
```

### Create Sub Account

#### Request

* path: /v1/sub/account/create
* account_char_str： the charset of sub-account name

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111"
  },
  "account": "",
  "sub_account_list": [
    {
      "account": "",
      "mint_for_account": "",
      "account_char_str": [
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        }
      ],
      "register_years": 1,
      "type": "blockchain",
      "key_info": {
        "coin_type": "60",
        "chain_id": "1",
        "key": "0x111..."
      }
    }
  ]
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "action": "create_sub_account",
    "sign_key": "",
    "list": [
      {
        "sign_list": [
          {
            "sign_type": 3,
            "sign_msg": "from did: 0x123"
          }
        ]
      }
    ]
  }
}
```

### Edit Sub Account

#### Request

* path: /v1/sub/account/edit
* params:
    * edit_key: owner, manager, records

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111..."
  },
  "account": "",
  "edit_key": "",
  "edit_value": {
    "owner": {
      "type": "blockchain",
      "key_info": {
        "coin_type": "60",
        "chain_id": "1",
        "key": "0x111..."
      }
    },
    "manager": {
      "type": "blockchain",
      "key_info": {
        "coin_type": "60",
        "chain_id": "1",
        "key": "0x111..."
      }
    },
    "records": [
      {
        "key": "",
        "type": "",
        "label": "",
        "value": "",
        "ttl": ""
      }
    ]
  }
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "action": "edit_sub_account",
    "sign_key": "",
    "list": [
      {
        "sign_list": [
          {
            "sign_type": 3,
            "sign_msg": "from did: 0x123"
          }
        ]
      }
    ]
  }
}
```

### Send Transaction

#### Request

* path: /v1/transaction/send

```json
{
  "action": "enable_sub_account",
  "sign_key": "",
  "list": [
    {
      "sign_list": [
        {
          "sign_type": 3,
          "sign_msg": "0x123"
        }
      ]
    }
  ]
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
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
    "chain_id": "1",
    "key": "0x111..."
  },
  "action": "enable_sub_account",
  "account": ""
}
```

#### Response

* status: 0: pending, 2: unsend

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "block_number": 0,
    "hash": "",
    "status": 0
  }
}
```

### Task Status

#### Request

* path: /v1/task/status

```json
{
  "task_id": "",
  "hash": ""
}
```

#### Response

* status: 0: pending, 1: ok, 2: fail

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "status": 0
  }
}
```

### Sub Account Mint Status

#### Request

* path: /v1/sub/account/mint/status

```json
{
  "sub_account": ""
}
```

#### Response

* status: 0: pending, 1: ok, 2: fail

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "status": 0
  }
}
```

### Custom Script Set

#### Request

* path: /v1/custom/script/set

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111..."
  },
  "account": "test.bit",
  "custom_script_args": "",
  "custom_script_config": {
    "1": {
      "new": 5000000,
      "renew": 5000000
    },
    "2": {
      "new": 1000000,
      "renew": 1000000
    }
  }
}
```

#### Response

```json
{
  "action": "config_sub_account_custom_script",
  "sign_key": "",
  "list": [
    {
      "sign_list": [
        {
          "sign_type": 3,
          "sign_msg": "0x123"
        }
      ]
    }
  ]
}
```

### Custom Script Info

#### Request

* path: /custom/script/info

```json
{
  "account": "test.bit"
}
```

#### Response

```json
{
  "custom_script_args": "",
  "custom_script_config": {
    "1": {
      "new": 5000000,
      "renew": 5000000
    },
    "2": {
      "new": 1000000,
      "renew": 1000000
    }
  }
}
```

### Custom Script Price

#### Request

* path: /custom/script/price

```json
{
  "sub_account": "123.test.bit"
}
```

#### Response

```json
{
  "custom_script_price": {
    "new": 1000000,
    "renew": 1000000
  }
}
```

### Owner Profit

#### Request

* path: /owner/profit

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111..."
  },
  "account": "tzh2022070601.bit"
}
```

#### Response

```json
{
  "owner_profit": "256.8"
}
```

### Profit Withdraw

#### Request

* path: /profit/withdraw

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111..."
  },
  "account": "tzh2022070601.bit"
}
```

#### Response

```json
{
  "hash": "0x00...",
  "action": "collect_sub_account_profit"
}
```

## INTERNAL API LIST

### Internal Mint Sub Account

#### Request

* path: /v1/internal/sub/account/mint

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "chain_id": "1",
    "key": "0x111"
  },
  "account": "",
  "sub_account_list": [
    {
      "account": "",
      "account_char_str": [
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        },
        {
          "char_set_name": 2,
          "char": "a"
        }
      ],
      "register_years": 1,
      "type": "blockchain",
      "key_info": {
        "coin_type": "60",
        "chain_id": "1",
        "key": "0x111..."
      }
    }
  ]
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {}
}
```

### Internal Check Smt Info

#### Request

* path: /v1/internal/smt/check

```json
{
  "parent_account_id": "",
  "limit": 1
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "list": [
      {
        "account_id": "",
        "chain_value": "",
        "smt_value": "",
        "diff": false
      }
    ]
  }
}
```

### Internal Update Smt

#### Request

* path: /v1/internal/smt/update

```json
{
  "parent_account_id": "",
  "sub_account_id": "",
  "value": ""
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "root": ""
  }
}
```
