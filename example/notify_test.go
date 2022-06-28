package example

import (
	"das_sub_account/notify"
	"fmt"
	"testing"
)

func TestSendNotifyDiscord(t *testing.T) {
	str := "** xxxx ** registered for 1 year(s)"
	webhook := ""
	fmt.Println(notify.SendNotifyDiscord(webhook, str))
}
