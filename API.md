* [API LIST](#api-list)
    * [Version](#version)
    * [Get Config Info](#get-config-info)
    * [Get Account List](#get-account-list)
    * [Get Account Detail](#get-account-detail)
    * [Get Sub Account List](#get-sub-account-list)
    * [Transaction Status](#transaction-status)
    * [Sub Account Mint Status](#sub-account-mint-status)
    * [Init Sub Account](#init-sub-account)
    * [Check Sub Account](#check-sub-account)
    * [Create Sub Account](#create-sub-account)
    * [Renew Sub Account](#Renew-Sub-Account)
    * [Check Renew Sub Account](#Check-Renew-Sub-Account)
    * [Edit Sub Account](#edit-sub-account)
    * [Send Transaction](#send-transaction)
    
    * [Task Status](#task-status)
   
    * [Custom Script Set](#custom-script-set)
    * [Custom Script Info](#custom-script-info)
    * [Custom Script Price](#custom-script-price)
    * [Owner Profit](#owner-profit)
    * [Profit Withdraw](#profit-withdraw)
* [INTERNAL API LIST](#internal-api-list)
    * [Internal Mint Sub Account](#internal-mint-sub-account)
    * [Internal Check Smt Info](#internal-check-smt-info)
    * [Internal Update Smt](#internal-update-smt)
    * [Internal Smt Info](#Internal-Smt-Info)
    * [Internal Smt SyncTree](#Internal-Smt-SyncTree)
    * [Owner Payment Export](Owner-Payment-Export)
    * [Internal Unipay Notice](#Internal-Unipay-Notice)
    * [Service Provider Withdraw](#Service Provider Withdraw)
    * [Service Provider Withdraw2](#Service Provider Withdraw2)
    * [Internal Recycle Account](Internal-Recycle-Account)
    * [Coupon Statistical Info](#Coupon-Statistical-Info)
    * [Padge Record Edit](#Padge-Record-Edit)
* [API for SubAccount Distribution](#API-for-SubAccount-Distribution)
  * [Coupon Order Info](#Coupon-Order-Info)
  * [Coupon Code Info](Coupon-Code-Info)
  * [Coupon Info](Coupon-Info)
  * [Coupon Download](Coupon-Download)
  * [Coupon Order Create](Coupon-Order-Create)
  * [Signin Info](Signin-Info)
  * [Statistical Info](#Statistical-Info)
  * [Distribution List](#Distribution-List)
  * [Update Mint Config](#Update-Mint-Config)
  * [Get Mint Config](#Get-Mint-Config)
  * [Search Account for Distribution](#Search-Account-for-Distribution)
  * [Create Order for Distribution](#Create-Order-for-Distribution)
  * [Return Order Pay Hash](#Return-Order-Pay-Hash)
  * [Get Order Info](#Get-Order-Info)
  * [Enable or Disable Distribution](#Enable-or-Disable-Distribution)
  * [Get Flag for Distribution](#Get-Flag-for-Distribution)
  * [Currency List](#Currency-List)
  * [Update Currency](#Update-Currency)
  * [Payment Record](#Payment-Record)
  * [Price Rule List](#Price-Rule-List)
  * [Update Price Rule](#Update-Price-Rule)
  * [Preserved Rule List](#Preserved-Rule-List)
  * [Update Preserved Rule](#Update-Preserved-Rule)
  * [Init SubAccount for Fee](#Init-SubAccount-for-Fee)
  * [API for Approval](APIApproval.md)
## API LIST

Please familiarize yourself with the meaning of some common parameters before reading the API list:

| param                                                                       | description                                        |
|:----------------------------------------------------------------------------|:---------------------------------------------------|
| type                                                                        | Filled with "blockchain" for now                   |
| [coin_type](https://github.com/satoshilabs/slips/blob/master/slip-0044.md)  | 60: eth, 195: trx, 9006: bsc, 966: matic, 3: doge  |
| account                                                                     | Contains the suffix `.bit` in it                   |
| key                                                                         | Generally refers to the blockchain address for now |

#### Version

**Request**

* path: /v1/version
* param: none

**Response**

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "version": 1.0
  }
}
```

**Usage**

```curl
curl -X POST http://127.0.0.1:8120/v1/config/info
```

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
    "sub_account_common_fee": 0,
    "ckb_quote": "",
    "auto_mint": {
      "payment_min_price": 0,
      "service_fee_ratio": ""
    },
    "mint_costs_manually": 0.00,
    "renew_costs_manually": 0.00,
    "management_times": 2,
    "stripe": {
      "premium_percentage": 0.00,
      "premium_base": 0.00
    },
    "token_list": [
      {
        "token_id": "ckb_ckb",
        "coin_type": "60",
        "symbol": "",
        "decimals": 8,
        "price": 0.00,
        "display_name": "",
        "icon": ""
      }
    ]
  }
}
```

### Get Account List

#### Request

* path: /v1/account/list
    * category: 
      * 1 : List of Main Accounts
      * 2 : List of Sub-Accounts
      * 6 : List of Main Accounts with the Sub-Account Function Enabled

```json
{
  "page": 1,
  "size": 100,
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0x111..."
  },
  "address": "",
  "category": 6,
  "keyword": ""
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "total": 1,
    "list": [
      {
        "account": "",
        "account_id": "",
        "owner": {
          "type": "blockchain",
          "key_info": {
            "coin_type": "60",
            "key": "0x111..."
          }
        },
        "manager": {
          "type": "blockchain",
          "key_info": {
            "coin_type": "60",
            "key": "0x111..."
          }
        },
        "registered_at": 0,
        "expired_at": 0,
        "status": 0,
        "enable_sub_account": 0,
        "renew_sub_account_price": 0,
        "nonce": 0,
        "avatar": ""
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
      "account_id": "",
      "owner": {
        "type": "blockchain",
        "key_info": {
          "coin_type": "60",
          "key": "0x111..."
        }
      },
      "manager": {
        "type": "blockchain",
        "key_info": {
          "coin_type": "60",
          "key": "0x111..."
        }
      },
      "registered_at": 0,
      "expired_at": 0,
      "status": 0,
      "enable_sub_account": 0,
      "renew_sub_account_price": 0,
      "nonce": 0,
      "avatar": ""
    },
    "records": [
      {
        "key": "",
        "type": "",
        "label": "",
        "value": "",
        "ttl": ""
      }
    ],
    "custom_script": ""
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
  "chain_type": 0,
  "address": "",
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
    "total": 1,
    "list": [
      {
        "account": "",
        "account_id": "",
        "owner": {
          "type": "blockchain",
          "key_info": {
            "coin_type": "60",
            "key": "0x111..."
          }
        },
        "manager": {
          "type": "blockchain",
          "key_info": {
            "coin_type": "60",
            "key": "0x111..."
          }
        },
        "registered_at": 0,
        "expired_at": 0,
        "status": 0,
        "enable_sub_account": 0,
        "renew_sub_account_price": 0,
        "nonce": 0,
        "avatar": ""
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
        "key": "0x111..."
      },
      "address": ""
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
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "from did: 0x123"
      }
    ]
  }
}
```

### Renew Sub Account

#### Request

* path: /v1/sub/account/renew
* account_char_str： the charset of sub-account name

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
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
        "key": "0x111..."
      },
      "address": ""
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
    "action": "edit_sub_account",
    "sign_key": "",
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "from did: 0x123"
      }
    ]
  }
}
```
### Check Renew Sub Account

#### Request

* path: /v1/sub/account/renew/check
* account_char_str： the charset of sub-account name

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
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
        "key": "0x111..."
      },
      "address": ""
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
    "action": "edit_sub_account",
    "sign_key": "",
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "from did: 0x123"
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
    "key": "0x111..."
  },
  "account": "",
  "edit_key": "",
  "edit_value": {
    "owner": {
      "type": "blockchain",
      "key_info": {
        "coin_type": "60",
        "key": "0x111..."
      }
    },
    "manager": {
      "type": "blockchain",
      "key_info": {
        "coin_type": "60",
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
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "from did: 0x123"
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
  "action": "create_approval",    // same with the api return
  "sub_action": "",               // same with the api return
  "is712": true,
  "sign_address": "0x111...",     // only sign_type='309' webauthn, You need to fill in this address
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
  "chain_type": 0,
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
  "sign_list": [
    {
      "sign_type": 3,
      "sign_msg": "0x123"
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
    "key": "0x111..."
  },
  "account": "tzh2022070601.bit"
}
```

#### Response

```json
{
  "owner_profit": "256.8",
  "bit_profit": ""
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
    "key": "0x111..."
  },
  "account": "tzh2022070601.bit",
  "is_withdraw_dot_bit": true
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

### Internal Smt Info

#### Request

* path: /v1/internal/smt/info

```json
{
  "parent_account_id": ""
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

### Internal Smt SyncTree

#### Request

* path: /v1/internal/smt/syncTree

```json
{
  "account_id": []
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

### Owner Payment Export

#### Request

* path: /v1/owner/payment/export

```json
{
  "account": "",
  "end": "",
  "payment": true
}
```

#### Response

### Internal Unipay Notice

#### Request

* path: /v1/unipay/notice

```json
{
  "business_id": "",
  "event_list": [
    {
      "event_type": "ORDER.PAY",
      "order_id": "",
      "pay_status": 0,
      "pay_hash": "",
      "pay_address": "",
      "refund_status": "",
      "refund_hash": ""
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
    "root": ""
  }
}
```

### Service Provider Withdraw

#### Request

* path: /v1/service/provider/withdraw

```json
{
  "service_provider_address": ""
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "hash": [""],
    "action": ""
  }
}
```

### Service Provider Withdraw2

#### Request

* path: /v1/service/provider/withdraw2

```json
{
  "service_provider_address": "",
  "account": "",
  "withdraw": true
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "hash": "",
    "amount": 0.00
  }
}
```

### Internal Recycle Account

#### Request

* path: /v1/internal/recycle/account

```json
{
  "sub_account_ids": [""]

}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": null
}
```

### Coupon Statistical Info

#### Request

* path: /v1/coupon/statistical/info

```json
{
}
```

#### Response

```json
{
  "errno": 0,
  "errmsg": "",
  "data": {
    "total": 0,
    "used": 0,
    "accounts": 0
  }
}
```
### Padge Record Edit

#### Request

* path: /v1/padge/record/edit

```json
{
  "list": [
    {
      "account": "",
      "nonce": 0,
      "signature": "",
      "sign_address": "",
      "expired_at": "",
      "alg_id": 0,
      "sub_alg_id": 0,
      "payload": "",
      "records": [
        {
          "key": "",
          "type": "",
          "label": "",
          "value": "",
          "ttl": 0
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
    "total": 0,
    "used": 0,
    "accounts": 0
  }
}
```


## API for SubAccount Distribution

### Statistical Info

#### Request

* path: /v1/statistical/info

```json
{
  "account": "test.bit"
}
```

#### Response

```json
{
  "err_no":0,
  "err_msg":"",
  "data":{
    "sub_account_num": 0, 
    "address_num": 0,     
    "income_info": [      
      {
        "type": "ETH",       
        "balance": "126560", 
        "total": "126560",
        "background_color": ""
      },
      {
        "type": "USDT-TRC20",
        "balance": "126560",
        "total": "126560",
        "background_color": ""
      }
    ],
    "ckb_spending":{      
      "balance": "12609", 
      "total": "12609"    
    },
    "dp_spending":{
      "balance": "12609",
      "total": "12609"
    },
    "auto_mint":{ 
      "enable": true,  
      "first_enable_time": 1683703195670 
    },
    "account_expired_at": 1715948028000 
  }
}
```

### Distribution List

#### Request

* path: /v1/distribution/list

```json
{
  "account": "test.bit",
  "page": 1,
  "size": 10
}
```

#### Response

```json
{
  "err_no":0,
  "err_msg":"",
  "data":{
    "page": 1,
    "total": 100,
    "list": [{
      "time": 1683599534179,
      "account": "test.bit",
      "years": 1,
      "amount": "100 USDT",
      "symbol": "",
      "action": "create",
      "coupon_info": {
        "cid": "",
        "order_amount": "",
        "set_name": "",
        "code": "",
        "coupon_price": "",
        "user_amount": ""
      }
    }]
  }
}
```

### Update Mint Config

#### Request

* path: /v1/mint/config/update

```json
{
  "type":"blockchain",
  "key_info":{
    "coin_type":"60",
    "key":"0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "account": "test.bit",
  "title": "",
  "desc": "",
  "benefits": "",
  "links": [
    {
      "app": "Twitter",
      "link": ""
    },
    {
      "app": "Telegram",
      "link": ""
    },
    {
      "app": "Website",
      "link": ""
    }
  ],
  "background_color": "",
  "timestamp": 1683547860 ,
  "mint_success_page": [{
    "type": "",
    "url": ""
  }]
}
```

#### Response

```json
{
  "err_no":0,
  "err_msg":"",
  "data": {
    "action":"Update-Mint-Config",
    "sub_action":"",
    "sign_key":"d395abc4037853fd5534f913ae8a6dd5",
    "sign_list":[
      {
        "sign_type":3,
        "sign_msg":"From .bit: 8b3a8750b3ded888c3b4ac53a80f7665e31ef6862e491bd634d78db4f6d25b9e"}
    ]
  }
}
```




### Get Mint Config

#### Request

* path: /v1/mint/config/get

```json
{
  "account": "test.bit"
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "title": "",
    "desc": "",
    "benefits": "",
    "links": [
      {
        "app": "Twitter",
        "link": ""
      },
      {
        "app": "Telegram",
        "link": ""
      },
      {
        "app": "Website",
        "link": ""
      }
    ],
    "background_color": "",
    "mint_success_page": [
      {
        "type": "",
        "url": ""
      }
    ]
  }
}
```
### Coupon Order Info
#### Request

* path: /v1/coupon/order/info

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0x111"
  },
  "account": "",
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "order_id": "",
    "token_id": "",
    "payment_address": "",
    "contract_address": "",
    "client_secret": "",
    "amount": 0.00,
    "pay_hash": "",
    "order_status": 0,
    "cid": ""
  }
}
```
### Coupon Code Info
#### Request

* path: /v1/coupon/order/info

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0x111"
  },
  "account": "",
  "cid": "",
  "page": 1,
  "page_size": 10
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "total": 0,
    "used": 0,
    "name": "",
    "note": "",
    "price": "",
    "begin_at": 0,
    "expired_at": 0,
    "created_at": 0,
    "list": [
      {
        "code": "",
        "used_by": "",
        "status": 0
      }
    ]
  }
}
```

### Coupon Info
#### Request

* path: /v1/coupon/info

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0x111"
  },
  "code": "",
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "code": 0,
    "price": 0,
    
    "begin_at": 0,
    "expired_at": 0,
    "status": 0
  }
}
```

### Coupon Download
#### Request

* path: /v1/coupon/download

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0x111"
  },
  "account": "",
  "cid": ""
}
```

#### Response
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": null
}
```


### Signin Info
#### Request

* path: /v1/signin/info

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0x111"
  },

}
```

#### Response
```json
{
  "err_no": 0,
  "err_msg": "",
  "data": null
}
```

### Search Account for Distribution

#### Request

* path: /v1/auto/account/search

```json
{
  "type":"blockchain",
  "key_info":{
    "coin_type":"60",
    "key":"0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "sub_account": "test.test.bit",
  "action_type": 0
}
```

#### Response

```json
{
  "err_no":0,
  "err_msg":"",
  "data": {
    "price": "100.00", 
    "max_year": 2, 
    "status": 0, 
    "is_self": false, 
    "order_id": "" ,
    "expired_at": 0,
    "premium_percentage": "0.036", // for usd premium
    "premium_base": "0.52" // for usd premium
    "default_renew_rule": true
  }
}
```


### Create Order for Distribution

#### Request

* path: /v1/auto/order/create
  * token_id
  * eth_eth
  * tron_trx
  * bsc_bnb
  * stripe_usd
  * tron_trc20_usdt
  * bsc_bep20_usdt
  * polygon_matic

```json
{
  "type":"blockchain",
  "key_info":{
    "coin_type":"60",
    "key":"0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "sub_account": "test.test.bit",  
  "action_type": 0,     
  "token_id": "eth_eth",  
  "years":1 ,
  "coupon_code": ""
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "order_id": "" ,
    "payment_address": "" ,
    "amount": "",
    "contract_address": "", // for usdt contract
    "client_secret": "", // for stripe usd
    "payment_status": 0
  }
}
```


### Return Order Pay Hash

#### Request

* path: /v1/auto/order/hash

```json
{
  "type":"blockchain",
  "key_info":{
    "coin_type":"60",
    "key":"0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "order_id": "", 
  "hash": ""  
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": null
}
```


### Get Order Info

#### Request

* path: /v1/auto/order/info

```json
{
  "type":"blockchain",
  "key_info":{
    "coin_type":"60",
    "chain_id":"",
    "key":"0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"
  },
  "order_id":"af7054eaf87de38a592bec32ff853fa6"
}
```

#### Response

```json
{
  "order_id":"af7054eaf87de38a592bec32ff853fa6",
  "token_id":"eth_erc20_usdt",
  "amount":"10045821",
  "pay_hash":"0x1a7cdadd9010cb03cc4a0d92af97ca0aac68ec25185f7e29610a67dd7f745f30",
  "order_status":5
}
```


### Enable or Disable Distribution

#### Request

* path: /v1/config/auto_mint/update

```json
{
  "type":"blockchain",
  "key_info":{
    "coin_type":"60",
    "key":"0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "account": "test.bit",
  "enable": true 
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "config_sub_account",
    "sub_action": "",
    "sign_key": "d4f3174152b63f51862d4684b1aba3b3",
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "From .bit: c9c151e30e4e071e84c06dd4419ff6da680f510f228c274929fddf5fcbd0e9d3"
      },
      {
        "sign_type": 0,
        "sign_msg": "0x03f4fa778587a862ff02c5e2f96a95e5d70b7a97a294102477e2c94c6baf5bee"
      }
    ]
  }
}
```


### Get Flag for Distribution

#### Request

* path: /v1/config/auto_mint/get

```json
{
  "account": "test.bit"
}
```

#### Response

```json
{
  "err_no":0,
  "err_msg":"",
  "data": {
    "enable": true 
  }
}
```


### Currency List

#### Request

* path: /v1/currency/list

```json
{
  "account": "test.bit"
}
```

#### Response

```json
{
  "err_no":0,
  "err_msg":"",
  "data": [
    {
      "token_id": "eth_eth",
      "enable": true,        
      "have_record": false ,  
      "symbol": "ETH",
      "price":"",
      "decimals":18
    },
    {
      "token_id": "tron_trx",
      "enable": true,
      "have_record": false,
      "symbol": "TRX",
      "price":"",
      "decimals":6
    },
    {
      "bsc_bnb": "bsc_bnb",
      "enable": true,
      "have_record": false,
      "symbol": "BNB",
      "price":"",
      "decimals":18
    }
  ]
}
```


### Update Currency

#### Request

* path: /v1/currency/update

```json
{
  "type":"blockchain",
  "key_info":{
    "coin_type":"60",
    "key":"0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "account": "test.bit",
  "token_id": "eth_eth",
  "enable": true,
  "timestamp": 1683547860
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "Update-Currency",
    "sub_action": "",
    "sign_key": "d395abc4037853fd5534f913ae8a6dd5",
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "From .bit: 8b3a8750b3ded888c3b4ac53a80f7665e31ef6862e491bd634d78db4f6d25b9e"
      }
    ]
  }
}
```

### Coupon Order Create

#### Request

* path: /v1/coupon/order/create

```json
{
  "type":"blockchain",
  "key_info":{
    "coin_type":"60",
    "key":"0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "order_id": "",
  "account": "test.bit",
  "token_id": "eth_eth",
  "num": 1,
  "name": "",
  "note": "",
  "price": "",
  "begin_at": 0,
  "expired_at": 0
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "order_id": "",
    "payment_address": "",
    "contract_address": "",
    "client_secret": "",
    "amount": 0.00
  }
}
```


### Payment Record

#### Request

* path: /v1/auto/payment/list

```json
{
  "account": "test.bit",
  "page": 1,
  "size": 10
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "total": 1,
    "list": [{
      "time": 1683599534179,
      "amount": "10.0 ETH"
    }]
  }
}
```


### Price Rule List

#### Request

* path: /v1/price/rule/list

```json
{
  "account": "test.bit"
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "list": [
      {
        "name": "emoji",
        "note": "",
        "price": 10000000,
        "ast": {
          "type": "function",
          "name": "include_chars",
          "arguments": [
            {
              "type": "variable",
              "name": "account_chars"
            },
            {
              "type": "value",
              "value_type": "string[]",
              "value": [
                "⚠️",
                "❌",
                "✅"
              ]
            }
          ]
        },
        "status": 1
      }
    ]
  }
}
```

### Update Price Rule

#### Request

* path: /v1/price/rule/update

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "account": "test.bit",
  "list": [
    {
      "index": 2,
      "name": "account length",
      "note": "",
      "price": 100000000,
      "ast": {
        "type": "operator",
        "symbol": "==",
        "expressions": [
          {
            "type": "variable",
            "name": "account_length"
          },
          {
            "type": "value",
            "value_type": "uint8",
            "value": 1
          }
        ]
      },
      "status": 1
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
    "action": "enable_sub_account",
    "sub_action": "",
    "sign_key": "d395abc4037853fd5534f913ae8a6dd5",
    "list": [
      {
        "sign_list": [
          {
            "sign_type": 3,
            "sign_msg": "From .bit: 8b3a8750b3ded888c3b4ac53a80f7665e31ef6862e491bd634d78db4f6d25b9e"
          }
        ]
      }
    ]
  }
}
```

### Preserved Rule List

#### Request

* path: /v1/preserved/rule/list

```json
{
  "account": "test.bit"
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "list": [
      {
        "name": "emoji",
        "note": "",
        "price": 10000000,
        "ast": {
          "type": "function",
          "name": "include_chars",
          "arguments": [
            {
              "type": "variable",
              "name": "account_chars"
            },
            {
              "type": "value",
              "value_type": "string[]",
              "value": [
                "⚠️",
                "❌",
                "✅"
              ]
            }
          ]
        },
        "status": 1
      }
    ]
  }
}
```

### Update Preserved Rule

#### Request

* path: /v1/preserved/rule/update

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "account": "test.bit",
  "list": [
    {
      "index": 2,
      "name": "account length",
      "note": "",
      "price": 100000000,
      "ast": {
        "type": "operator",
        "symbol": "==",
        "expressions": [
          {
            "type": "variable",
            "name": "account_length"
          },
          {
            "type": "value",
            "value_type": "uint8",
            "value": 1
          }
        ]
      },
      "status": 1
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
    "action": "enable_sub_account",
    "sub_action": "",
    "sign_key": "d395abc4037853fd5534f913ae8a6dd5",
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "From .bit: 8b3a8750b3ded888c3b4ac53a80f7665e31ef6862e491bd634d78db4f6d25b9e"
      }
    ]
  }
}
```

### Init SubAccount for Fee

#### Request

* path: /v1/sub/account/init/free

```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0xc9f53b1d85356b60453f867610888d89a0b667ad"
  },
  "account": "test.bit",
  "address": ""
}
```

#### Response

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "action": "enable_sub_account",
    "sub_action": "",
    "sign_key": "d395abc4037853fd5534f913ae8a6dd5",
    "sign_list": [
      {
        "sign_type": 3,
        "sign_msg": "From .bit: 8b3a8750b3ded888c3b4ac53a80f7665e31ef6862e491bd634d78db4f6d25b9e"
      }
    ]
  }
}
```