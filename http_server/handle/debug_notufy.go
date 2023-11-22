package handle

import "github.com/gin-gonic/gin"

func (h *HttpHandle) DebugNotify(ctx *gin.Context) {
	h.TxTool.Metrics.ErrNotify().WithLabelValues("title", "test").Inc()
}
