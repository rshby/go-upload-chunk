package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"go-upload-chunk/server/config"
	"go-upload-chunk/server/internal/entity"
	"go-upload-chunk/server/internal/utils"
	"io"
	"os"
)

type fileService struct {
	validate *validator.Validate
}

func (f *fileService) UploadChunk(ctx context.Context, request entity.UploadChunkRequestServiceDTO) error {
	logger := logrus.WithContext(ctx)

	// validate request
	if err := f.validate.Struct(request); err != nil {
		logger.Error(err)
		return err
	}

	var (
		requestHeader = request.RequestHeader
		content       = request.Content
	)

	// check local folder chunk
	if err := f.CheckAndCreateFolder(config.FolderUploadChunk()); err != nil {
		logger.Error(err)
		return err
	}

	// check file chunk if it isn't exists
	filepath := fmt.Sprintf("%s/%s-chunk-%d", config.FolderUploadChunk(), requestHeader.Filename, requestHeader.ChunkIndex)
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		newFile, err := os.Create(filepath)
		if err != nil {
			logger.Error(err)
			return err
		}

		defer newFile.Close()

		if _, err = newFile.Write(content.Bytes()); err != nil {
			logger.Error(err)
			return nil
		}
	}

	// file chunk already exists, then check total chunk file
	var totalChunkFiles int

	// if total files number is same as we expect, then combine mutiple chunk into a one file
	if totalChunkFiles == requestHeader.TotalChunk {
		// looping each file -> combine multiple files into a one file
		// create new file in folder upload/final
		fp := fmt.Sprintf("%s/%s", config.FolderUploadChunk(), requestHeader.Filename)
		finalFile, err := os.Create(fp)
		if err != nil {
			logger.Error(err)
			return err
		}

		defer finalFile.Close()

		for i := 0; i < totalChunkFiles; i++ {
			// open file
			fileChunkOneFilePath := fmt.Sprintf("%s/%s-chunk-%d", config.FolderUploadChunk(), requestHeader.Filename, i)
			fileChunkOneFile, err := os.Open(fileChunkOneFilePath)
			if err != nil {
				logger.Error(err)
				return err
			}

			// get buffer from Pool
			bufOneChunk := utils.ByteBufferPool.Get().(*bytes.Buffer)
			if _, err := io.Copy(bufOneChunk, fileChunkOneFile); err != nil {
				logger.Error(err)

				// reset buffer
				bufOneChunk.Reset()

				// put back buffer to Pool
				utils.ByteBufferPool.Put(bufOneChunk)
				return err
			}

			if _, err := finalFile.Write(bufOneChunk.Bytes()); err != nil {
				logger.Error(err)
				return err
			}

			// don't forget to put back buffer to Pool
			bufOneChunk.Reset()

			// don't forget to close file on chunk
			_ = fileChunkOneFile.Close()
		}
	}

	return nil
}

func (f *fileService) CheckAndCreateFolder(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// create folder
		if err = os.Mkdir(path, os.ModePerm); err != nil {
			logrus.Error(err)
			return err
		}
	}

	return nil
}
