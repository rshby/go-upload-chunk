package router

import (
	"github.com/go-playground/validator/v10"
	"go-upload-chunk/server/internal/controller"
	"go-upload-chunk/server/internal/service"
)

func InitFileController(validate *validator.Validate) *controller.FileController {
	fileService := service.NewFileService(validate)
	return controller.NewFileController(fileService)
}
