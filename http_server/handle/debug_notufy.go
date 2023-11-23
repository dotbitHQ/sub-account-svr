package handle

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"time"
)

func (h *HttpHandle) DebugNotify(ctx *gin.Context) {
	h.TxTool.Metrics.ErrNotify().WithLabelValues(fmt.Sprint(time.Now().Unix()), "test").Inc()
}
