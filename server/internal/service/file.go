package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"go-upload-chunk/server/config"
	"go-upload-chunk/server/internal/entity"
	"go-upload-chunk/server/internal/utils"
	"io"
	"os"
	"path/filepath"
	_ "path/filepath"
)

type fileService struct {
	validate *validator.Validate
}

// UploadChunk uploads one chunk file, combines to one file
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

	// check file chunk if it isn't exists then create new file chunk
	filePath := fmt.Sprintf("%s/%s-chunk-%d", config.FolderUploadChunk(), requestHeader.Filename, requestHeader.ChunkIndex)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		var newChunkFile *os.File

		// create new chunk file
		newChunkFile, err = os.Create(filePath)
		if err != nil {
			logger.Error(err)
			return err
		}

		// write content to new chunk files
		if _, err = newChunkFile.Write(content.Bytes()); err != nil {
			logger.Error(err)

			// don't forget to close chunk
			_ = newChunkFile.Close()

			return nil
		}

		// don't forget to close chunk
		_ = newChunkFile.Close()
	}

	// file chunk already exists, then check total chunk file
	filePathPrefix := fmt.Sprintf("%s/%s-chunk-", config.FolderUploadChunk(), requestHeader.Filename)
	var totalChunkFiles int
	matchFiles, err := filepath.Glob(filePathPrefix)
	if err != nil {
		logger.Error(err)
		return err
	}

	// count total chunk files in local folder upload
	for _, _ = range matchFiles {
		totalChunkFiles++
	}

	// if total files number is same as we expect, then combine mutiple chunk into a one file
	if totalChunkFiles == requestHeader.TotalChunk {
		// create new file in folder upload/final
		fp := fmt.Sprintf("%s/%s", config.FolderUploadChunk(), requestHeader.Filename)
		finalFile, err := os.Create(fp)
		if err != nil {
			logger.Error(err)
			return err
		}

		defer finalFile.Close()

		// looping each chunk file, combine to final file
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
			if _, err = io.Copy(bufOneChunk, fileChunkOneFile); err != nil {
				logger.Error(err)

				// reset buffer
				bufOneChunk.Reset()

				// put back buffer to Pool
				utils.ByteBufferPool.Put(bufOneChunk)
				return err
			}

			// write from buffer to final file
			if _, err = finalFile.Write(bufOneChunk.Bytes()); err != nil {
				logger.Error(err)
				return err
			}

			// don't forget to put back buffer to Pool
			bufOneChunk.Reset()

			// don't forget to close file on chunk
			_ = fileChunkOneFile.Close()
		}

		logger.Info("success combine fine")

		// validate check sum from final file
		h := sha256.New()
		if _, err = io.Copy(h, finalFile); err != nil {
			logger.Error(err)
			return err
		}

		// compare checksum
		checkSumFinalFile := fmt.Sprintf("%x", h.Sum(nil))
		if checkSumFinalFile != requestHeader.CheckSum {
			return fmt.Errorf("invalid checksum")
		}

		// upload to Cloud
		logger.Info("process upload final file to cloud")
		logger.Info("success upload final file to cloud")
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
