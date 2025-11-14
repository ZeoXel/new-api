package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetVideoRouter(router *gin.Engine) {
	videoV1Router := router.Group("/v1")
	videoV1Router.GET("/videos/:task_id/content", controller.VideoProxy)
	videoV1Router.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		videoV1Router.POST("/video/generations", controller.RelayTask)
		videoV1Router.GET("/video/generations/:task_id", controller.RelayTask)
	}
	// openai compatible API video routes
	// docs: https://platform.openai.com/docs/api-reference/videos/create

	// Sora 视频生成路由 - 支持 multipart/form-data 文件上传
	// 适用于 sora-2, sora-2-pro 等模型
	// 示例请求: POST /v1/videos
	// 支持参数: model, prompt, size, input_reference (文件), seconds, watermark 等
	soraVideosRouter := router.Group("/v1")
	// 仿照 Kling，中间件顺序：SoraRequestConvert -> TokenAuth -> Distribute
	soraVideosRouter.Use(middleware.SoraRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		soraVideosRouter.POST("/videos", controller.RelayBltcy)
		soraVideosRouter.GET("/videos/:id", controller.RelayBltcy)
	}

	klingV1Router := router.Group("/kling/v1")
	klingV1Router.Use(middleware.KlingRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		klingV1Router.POST("/videos/text2video", controller.RelayTask)
		klingV1Router.POST("/videos/image2video", controller.RelayTask)
		klingV1Router.GET("/videos/text2video/:task_id", controller.RelayTask)
		klingV1Router.GET("/videos/image2video/:task_id", controller.RelayTask)
	}

	// Jimeng official API routes - direct mapping to official API format
	jimengOfficialGroup := router.Group("jimeng")
	jimengOfficialGroup.Use(middleware.JimengRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		// Maps to: /?Action=CVSync2AsyncSubmitTask&Version=2022-08-31 and /?Action=CVSync2AsyncGetResult&Version=2022-08-31
		jimengOfficialGroup.POST("/", controller.RelayTask)
	}
}
