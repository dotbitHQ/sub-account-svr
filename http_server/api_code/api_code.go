package api_code

type ApiCode = int

const (
	ApiCodeSuccess        ApiCode = 0
	ApiCodeError500       ApiCode = 500
	ApiCodeParamsInvalid  ApiCode = 10000
	ApiCodeMethodNotExist ApiCode = 10001
	ApiCodeDbError        ApiCode = 10002
	ApiCodeCacheError     ApiCode = 10003

	ApiCodeTransactionNotExist          ApiCode = 11001
	ApiCodeInsufficientBalance          ApiCode = 11007
	ApiCodeTxExpired                    ApiCode = 11008
	ApiCodeRejectedOutPoint             ApiCode = 11011
	ApiCodeSyncBlockNumber              ApiCode = 11012
	ApiCodeNotEnoughChange              ApiCode = 11014
	ApiCodeAccountNotExist              ApiCode = 30003
	ApiCodeAccountIsExpired             ApiCode = 30010
	ApiCodePermissionDenied             ApiCode = 30011
	ApiCodeSystemUpgrade                ApiCode = 30019
	ApiCodeRecordInvalid                ApiCode = 30020
	ApiCodeRecordsTotalLengthExceeded   ApiCode = 30021
	ApiCodeSameLock                     ApiCode = 30023
	ApiCodeAccountStatusOnSaleOrAuction ApiCode = 30031
	ApiCodeOnCross                      ApiCode = 30035
	ApiCodeParentAccountExpired         ApiCode = 30036

	ApiCodeEnableSubAccountIsOn               ApiCode = 40000
	ApiCodeNotExistEditKey                    ApiCode = 40001
	ApiCodeNotExistConfirmAction              ApiCode = 40002
	ApiCodeSignError                          ApiCode = 40003
	ApiCodeNotExistSignType                   ApiCode = 40004
	ApiCodeNotSubAccount                      ApiCode = 40005
	ApiCodeEnableSubAccountIsOff              ApiCode = 40006
	ApiCodeCreateListCheckFail                ApiCode = 40007
	ApiCodeTaskInProgress                     ApiCode = 40008
	ApiCodeDistributedLockPreemption          ApiCode = 40009
	ApiCodeRecordDoing                        ApiCode = 40010
	ApiCodeUnableInit                         ApiCode = 40011
	ApiCodeNotHaveManagementPermission        ApiCode = 40012
	ApiCodeSmtDiff                            ApiCode = 40013
	ApiCodeSuspendOperation                   ApiCode = 40014
	ApiCodeTaskNotExist                       ApiCode = 40015
	ApiCodeSameCustomScript                   ApiCode = 40016
	ApiCodeNotExistCustomScriptConfigPrice    ApiCode = 40017
	ApiCodeCustomScriptSet                    ApiCode = 40018
	ApiCodeProfitNotEnough                    ApiCode = 40019
	ApiCodeNoSupportPaymentToken              ApiCode = 40020
	ApiCodeOrderNotExist                      ApiCode = 40021
	ApiCodeRuleDataErr                        ApiCode = 40022
	ApiCodeParentAccountNotExist              ApiCode = 40023
	ApiCodeSubAccountMinting                  ApiCode = 40024
	ApiCodeSubAccountMinted                   ApiCode = 40025
	ApiCodeBeyondMaxYears                     ApiCode = 40026
	ApiCodeHitBlacklist                       ApiCode = 40027
	ApiCodeNoTSetRules                        ApiCode = 40028
	ApiCodeTokenIdNotSupported                ApiCode = 40029
	ApiCodeNoSubAccountDistributionPermission ApiCode = 40030
	ApiCodeSubAccountNoEnable                 ApiCode = 40031
	ApiCodeAutoDistributionClosed             ApiCode = 40032
	ApiCodeAccountCanNotBeEmpty               ApiCode = 40033
	ApiCodePriceRulePriceNotBeLessThanMin     ApiCode = 40034
	ApiCodePriceMostReserveTwoDecimal         ApiCode = 40035
	ApiCodeConfigSubAccountPending            ApiCode = 40036
	ApiCodeAccountRepeat                      ApiCode = 40037
	ApiCodeInListMostBeLessThan1000           ApiCode = 40038
	ApiCodePreservedRulesMostBeOne            ApiCode = 40039
	ApiCodeRuleSizeExceedsLimit               ApiCode = 40040
)

const (
	TextSystemUpgrade = "The service is under maintenance, please try again later."
)

type ApiResp struct {
	ErrNo  ApiCode     `json:"err_no"`
	ErrMsg string      `json:"err_msg"`
	Data   interface{} `json:"data"`
}

func (a *ApiResp) ApiRespErr(errNo ApiCode, errMsg string) {
	a.ErrNo = errNo
	a.ErrMsg = errMsg
}

func (a *ApiResp) ApiRespOK(data interface{}) {
	a.ErrNo = ApiCodeSuccess
	a.Data = data
}

func ApiRespErr(errNo ApiCode, errMsg string) ApiResp {
	return ApiResp{
		ErrNo:  errNo,
		ErrMsg: errMsg,
		Data:   nil,
	}
}
