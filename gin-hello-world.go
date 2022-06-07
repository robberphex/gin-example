package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

const AppName = "example-gin-opensergo"
const Port = 8080

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello world")
	})

	//go func() {
	//	for {
	//		routesInfo := r.Routes()
	//		processServiceContract(routesInfo, []string{"1.1.1.1"}, make(chan struct{}))
	//	}
	//}()

	r.Run(":" + strconv.Itoa(Port))
}
