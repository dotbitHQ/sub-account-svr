package notify

import (
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/parnurzeal/gorequest"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	log           = logger.NewLogger("notify", logger.LevelDebug)
	counterNotify = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "notify",
	}, []string{"title", "text"})
)

const (
	LarkNotifyUrl = "https://open.larksuite.com/open-apis/bot/v2/hook/%s"
)

func init() {
	txtool.PromRegister.MustRegister(counterNotify)
}

type MsgContent struct {
	Tag      string `json:"tag"`
	UnEscape bool   `json:"un_escape"`
	Text     string `json:"text"`
}
type MsgData struct {
	Email   string `json:"email"`
	MsgType string `json:"msg_type"`
	Content struct {
		Post struct {
			ZhCn struct {
				Title   string         `json:"title"`
				Content [][]MsgContent `json:"content"`
			} `json:"zh_cn"`
		} `json:"post"`
	} `json:"content"`
}

func SendLarkTextNotify(key, title, text string) {
	SendLarkTextNotifyWithSvr(key, title, text, true)
}

func SendLarkErrNotify(title, text string) {
	if title == "" || text == "" {
		return
	}
	counterNotify.WithLabelValues(title, text).Inc()
}

func SendLarkTextNotifyWithSvr(key, title, text string, withSvr bool) {
	if key == "" || text == "" {
		return
	}
	var data MsgData
	data.Email = ""
	data.MsgType = "post"
	if withSvr {
		data.Content.Post.ZhCn.Title = fmt.Sprintf("sub-account-svr: %s", title)
	} else {
		data.Content.Post.ZhCn.Title = title
	}
	data.Content.Post.ZhCn.Content = [][]MsgContent{
		{
			MsgContent{
				Tag:      "text",
				UnEscape: false,
				Text:     text,
			},
		},
	}
	url := fmt.Sprintf(LarkNotifyUrl, key)
	_, body, errs := gorequest.New().Post(url).Timeout(time.Second * 10).SendStruct(&data).End()
	if len(errs) > 0 {
		log.Error("sendLarkTextNotify req err:", errs)
	} else {
		log.Info("sendLarkTextNotify req:", body)
	}
}

func GetLarkTextNotifyStr(funcName, keyInfo, errInfo string) string {
	msg := fmt.Sprintf(`func：%s
key：%s
error：%s`, funcName, keyInfo, errInfo)
	return msg
}
