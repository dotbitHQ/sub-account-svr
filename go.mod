module das_sub_account

go 1.15

require (
	github.com/DeAccountSystems/das-lib v0.0.0-20220428061712-5460703d9be4
	github.com/elazarl/goproxy v0.0.0-20220115173737-adb46da277ac // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gin-gonic/gin v1.7.2
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/nervosnetwork/ckb-sdk-go v0.101.3
	github.com/parnurzeal/gorequest v0.2.16
	github.com/scorpiotzh/mylog v1.0.9
	github.com/scorpiotzh/toolib v1.1.3
	github.com/urfave/cli/v2 v2.3.0
	go.mongodb.org/mongo-driver v1.8.3
	gorm.io/gorm v1.22.1
	moul.io/http2curl v1.0.0 // indirect
)

replace github.com/ethereum/go-ethereum v1.9.14 => github.com/ethereum/go-ethereum v1.10.17
