module das_sub_account

go 1.16

require (
	github.com/dotbitHQ/das-lib v1.0.1-0.20230105081132-fa57137bed84
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gin-gonic/gin v1.7.2
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/nervosnetwork/ckb-sdk-go v0.101.3
	github.com/parnurzeal/gorequest v0.2.16
	github.com/scorpiotzh/mylog v1.0.10
	github.com/scorpiotzh/toolib v1.1.5
	github.com/shopspring/decimal v1.3.1
	github.com/urfave/cli/v2 v2.3.0
	go.mongodb.org/mongo-driver v1.9.1
	gorm.io/gorm v1.23.6
)

replace github.com/ethereum/go-ethereum v1.9.14 => github.com/ethereum/go-ethereum v1.10.17
