package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
		requestHeader   = request.RequestHeader
		totalChunkFiles int
	)

	hashChecksum := sha256.New()
	content := io.TeeReader(request.Content, hashChecksum)

	// get buffer from Pool
	buf := utils.ByteBufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		utils.ByteBufferPool.Put(buf)
	}()

	// copy from content to buf. content will be empty
	if _, err := io.Copy(buf, content); err != nil {
		logger.Error(err)
		return err
	}

	// copy from buf to request.Content, buf will be empty
	if _, err := io.Copy(request.Content, buf); err != nil {
		logger.Error(err)
		return err
	}

	// validate checksum
	checksum := hex.EncodeToString(hashChecksum.Sum(nil))
	if checksum != requestHeader.CheckSum {
		err := fmt.Errorf("invalid checksum ‼️")
		logger.Error(err)
		return err
	}

	// check local folder chunk
	if err := f.CheckAndCreateFolder(ctx, config.FolderUploadChunk()); err != nil {
		logger.Error(err)
		return err
	}

	// check file chunk if already exists
	filePath := fmt.Sprintf("%s/%s-chunk-%d", config.FolderUploadChunk(), requestHeader.Filename, requestHeader.ChunkIndex)
	if _, err := os.Stat(filePath); err == nil {
		logger.Infof("chuck file already exists 📩")
		return nil
	}

	// create new chunk file
	if err := f.CreateChunkFile(ctx, request); err != nil {
		logger.Error(err)
		return err
	}

	// find all files by given prefix path
	filePathPrefix := fmt.Sprintf("%s/%s-chunk-*", config.FolderUploadChunk(), requestHeader.Filename)
	matchFiles, err := filepath.Glob(filePathPrefix)
	if err != nil {
		logger.Error(err)
		return err
	}

	// count total chunk files
	for _, _ = range matchFiles {
		totalChunkFiles++
	}

	// if total files number is same as we expect, then combine mutiple chunk into a one file
	if totalChunkFiles == requestHeader.TotalChunk {
		if err = f.CreateFinalFile(ctx, request); err != nil {
			logger.Error(err)
			return err
		}
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

// CreateChunkFile creates new chunk file
func (f *fileService) CreateChunkFile(ctx context.Context, request entity.UploadChunkRequestServiceDTO) error {
	ctx, span := gootel.RecordSpan(ctx)
	defer span.End()

	logger := logrus.WithContext(ctx)

	// create new chunk file
	chunkFilePath := fmt.Sprintf("%s/%s-chunk-%d", config.FolderUploadChunk(), request.RequestHeader.Filename, request.RequestHeader.ChunkIndex)
	chunkFile, err := os.Create(chunkFilePath)
	if err != nil {
		logger.Error(err)
		return err
	}

	// don't forget to close chunk file at the end
	defer chunkFile.Close()

	// write content to chunk file
	if _, err = chunkFile.Write(request.Content.Bytes()); err != nil {
		logger.Error(err)
		return err
	}

	// sync chunk file
	if err = chunkFile.Sync(); err != nil {
		logger.Error(err)
		return err
	}

	logger.Infof("success create chunk file [%s] 🗳️", chunkFilePath)
	return nil
}

// CreateFinalFile creates new final file and combine from multiple chunk files into one final file
func (f *fileService) CreateFinalFile(ctx context.Context, request entity.UploadChunkRequestServiceDTO) error {
	ctx, span := gootel.RecordSpan(ctx)
	defer span.End()

	logger := logrus.WithContext(ctx)

	// check folder final
	if err := f.CheckAndCreateFolder(ctx, config.FolderUploadFinal()); err != nil {
		logger.Error(err)
		return err
	}

	finalFilePath := fmt.Sprintf("%s/%s", config.FolderUploadFinal(), request.RequestHeader.Filename)
	if _, err := os.Stat(finalFilePath); err == nil {
		logrus.Infof("final file already exists 📩")
		return nil
	}

	// create new final file
	finalFile, err := os.Create(finalFilePath)
	if err != nil {
		logger.Error(err)
		return err
	}

	// don't forget to close file at the end
	defer finalFile.Close()

	// combine from multiple chunk files into one final file
	if err = f.CombineChunkFiles(ctx, request, finalFile); err != nil {
		logger.Error(err)
		return err
	}

	logger.Infof("success create final file [%s] ✅", finalFilePath)
	return nil
}

// CombineChunkFiles combines multiple chunk files into one final file
func (f *fileService) CombineChunkFiles(ctx context.Context, request entity.UploadChunkRequestServiceDTO, finalFile *os.File) error {
	ctx, span := gootel.RecordSpan(ctx)
	defer span.End()

	logger := logrus.WithContext(ctx)

	// looping each chunk files
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

		// remove chunk file after append to final file
		if err = os.RemoveAll(chunkFilePath); err != nil {
			logger.Error(err)
			return err
		}
	}
	return nil
}
