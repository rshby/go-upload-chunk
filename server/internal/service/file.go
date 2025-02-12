package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	gootel "github.com/erajayatech/go-opentelemetry/v2"
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

// NewFileService creates new instance of fileService. it implements from interface FileService
func NewFileService(validate *validator.Validate) entity.FileService {
	return &fileService{validate}
}

// UploadChunk uploads one chunk file, combines to one file
func (f *fileService) UploadChunk(ctx context.Context, request entity.UploadChunkRequestServiceDTO) error {
	ctx, span := gootel.RecordSpan(ctx)
	defer span.End()

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
	if err := f.CheckAndCreateFolder(ctx, config.FolderUploadChunk()); err != nil {
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
	filePathPrefix := fmt.Sprintf("%s/%s-chunk-*", config.FolderUploadChunk(), requestHeader.Filename)
	var totalChunkFiles int

	logger.Infof("filepath chunk : [%s]", filePathPrefix)
	matchFiles, err := filepath.Glob(filePathPrefix)
	if err != nil {
		logger.Error(err)
		return err
	}

	// count total chunk files in local folder upload
	for _, _ = range matchFiles {
		totalChunkFiles++
	}

	logger.Infof("total chunk files : %d", totalChunkFiles)

	// if total files number is same as we expect, then combine mutiple chunk into a one file
	if totalChunkFiles == requestHeader.TotalChunk {
		// check folder final
		if err = f.CheckAndCreateFolder(ctx, config.FolderUploadFinal()); err != nil {
			logger.Error(err)
			return err
		}

		// create new final file inside folder ./upload/final
		fp := fmt.Sprintf("%s/%s", config.FolderUploadFinal(), requestHeader.Filename)
		finalFile, err := os.Create(fp)
		if err != nil {
			logger.Error(err)
			return err
		}

		// don't forget to close file at the end
		defer finalFile.Close()

		// combine multiple chunk files into one final file
		if err = f.CombineChunkFiles(ctx, request, finalFile); err != nil {
			logger.Error(err)
			return err
		}

		logger.Info("success combine files")

		// validate check sum from final file
		if err = finalFile.Sync(); err != nil {
			logger.Error(err)
			return err
		}

		fileInfo, err := finalFile.Stat()
		if err != nil {
			logger.Error(err)
			return err
		}

		logrus.Infof("final file size : %d 🗂️", fileInfo.Size())

		h := sha256.New()
		if _, err = io.Copy(h, finalFile); err != nil {
			logger.Error(err)
			return err
		}

		// compare checksum
		checkSumFinalFile := fmt.Sprintf("%x", h.Sum(nil))
		if checkSumFinalFile != requestHeader.CheckSum {
			logger.Infof("checksum from client : [%s]", requestHeader.CheckSum)
			logger.Infof("checksum from server : [%s]", checkSumFinalFile)
			_ = os.RemoveAll(fp)
			return fmt.Errorf("invalid checksum")
		}

		// upload to Cloud
		logger.Info("process upload final file to cloud")
		logger.Info("success upload final file to cloud")
	}

	return nil
}

// CheckAndCreateFolder checks folder, if not exists then create folder
func (f *fileService) CheckAndCreateFolder(ctx context.Context, path string) error {
	ctx, span := gootel.RecordSpan(ctx)
	defer span.End()

	logger := logrus.WithContext(ctx)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// create folder
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			logrus.Error(err)
			return err
		}

		logger.Infof("create new folder [%s]", path)
	}

	return nil
}

// CombineChunkFiles combines multiple chunk files into one final file
func (f *fileService) CombineChunkFiles(ctx context.Context, request entity.UploadChunkRequestServiceDTO, finalFile *os.File) error {
	ctx, span := gootel.RecordSpan(ctx)
	defer span.End()

	logger := logrus.WithContext(ctx)

	// looping each chunk file
	for i := 0; i < request.RequestHeader.TotalChunk; i++ {
		// open file chunk
		chunkFilePath := fmt.Sprintf("%s/%s-chunk-%d", config.FolderUploadChunk(), request.RequestHeader.Filename, i)
		oneChunkFile, err := os.Open(chunkFilePath)
		if err != nil {
			logger.Error(err)
			return err
		}

		logger.Infof("success open chunk file %s", chunkFilePath)

		// get buffer from Pool
		buf := utils.ByteBufferPool.Get().(*bytes.Buffer)

		// copy content from chunk file to buffer
		if _, err = io.Copy(buf, oneChunkFile); err == nil {
			// write to final file
			if _, err = finalFile.Write(buf.Bytes()); err == nil {
				logger.Infof("success append chunk file [%s] to final file", chunkFilePath)
			} else {
				logger.Error(err)
			}
		} else {
			logger.Error(err)
		}

		buf.Reset()
		utils.ByteBufferPool.Put(buf)
		_ = oneChunkFile.Close()

		_ = os.RemoveAll(chunkFilePath)

		// if any error
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	return nil
}
