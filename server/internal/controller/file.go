package controller

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go-upload-chunk/server/internal/entity"
	"go-upload-chunk/server/internal/utils"
	"io"
)

type FileController struct {
	fileService entity.FileService
}

// UploadChunk uploads file for chunks
func (f *FileController) UploadChunk(c *gin.Context) {
	logger := logrus.WithContext(c)

	// get request body (binary)
	buf := utils.ByteBufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		utils.ByteBufferPool.Put(buf)
	}()

	// copy from request body to buffer
	if _, err := io.Copy(buf, c.Request.Body); err != nil {
		logger.Error(err)
		return
	}

	// call method in service
	if err := f.fileService.UploadChunk(c.Request.Context(), entity.UploadChunkRequestServiceDTO{
		RequestHeader: new(entity.RequestHeaderDTO).Header(c),
		Content:       buf,
	}); err != nil {
		logger.Error(err)
		return
	}

	// success upload chunk
}
