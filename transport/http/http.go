package http

import (
	"github.com/gin-gonic/gin"
	"github.com/nojyerac/semaphore/data"
)

func RegisterRoutes(src *data.Source, router gin.IRouter) {
	_ = router
	_ = src
}
