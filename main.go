package main

import (
	"Robin-Camp/api"
	"Robin-Camp/internal"
	"context"
	"flag"
	"log"
	"os"

	_ "Robin-Camp/docs"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/swagger"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
)

func main() {
	// 从 .env 文件加载环境变量
	err := godotenv.Load()
	if err != nil {
		log.Println("加载 .env 文件失败, 将使用系统环境变量")
	}
	internal.InitDB()

	port := flag.String("p", "8080", "监听端口")
	address := flag.String("a", "0.0.0.0", "监听地址")
	help := flag.Bool("h", false, "显示帮助")
	swaggerFlag := flag.Bool("swagger", false, "启用Swagger文档")

	// 从.env加载配置
	envPort := os.Getenv("PORT")
	if envPort != "" {
		*port = envPort
	}
	envAddress := os.Getenv("ADDRESS")
	if envAddress != "" {
		*address = envAddress
	}

	flag.Parse()
	if *help {
		flag.Usage()
		return
	}
	h := server.Default(server.WithHostPorts(*address + ":" + *port))

	apiRoute := h.Group("/")
	// 注册认证路由 (公开)
	api.RegisterRoutes(apiRoute)

	// 404 handler
	h.NoRoute(func(ctx context.Context, c *app.RequestContext) {
		c.JSON(404, map[string]interface{}{"message": "404 Not Found"})
	})

	if *swaggerFlag {
		hlog.Info("Swagger文档已启用，访问 http://" + *address + ":" + *port + "/swagger/index.html 查看")
		url := swagger.URL("http://" + *address + ":" + *port + "/swagger/doc.json")
		h.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler, url))
	}
	h.Spin()
}
