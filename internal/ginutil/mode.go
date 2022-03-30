package ginutil

import (
	"os"

	"github.com/gin-gonic/gin"
)

// SetMode from the GIN_MODE environment variable. Unlike the init function
// in gin, this function defaults to ReleaseMode when the environment variable
// has no value.
func SetMode() {
	mode := os.Getenv(gin.EnvGinMode)
	if mode == "" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)
}
