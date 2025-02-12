package router

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func SetupRouter(app *gin.RouterGroup, validate *validator.Validate) {
	// init dependency injection
	fileController := InitFileController(validate)

	apiV1 := app.Group("v1")
	{
		// upload file chunk
		fileGroup := apiV1.Group("file")
		{
			fileGroup.POST("/chunk", fileController.UploadChunk)
		}
	}
}
