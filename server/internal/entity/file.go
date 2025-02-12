package entity

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"strconv"
)

type FileService interface {
	UploadChunk(ctx context.Context, request UploadChunkRequestServiceDTO) error
}

type RequestHeaderDTO struct {
	Filename   string `json:"filename" validate:"required"`
	CheckSum   string `json:"check_sum" validate:"required"`
	ChunkIndex int    `json:"chunk_index"`
	TotalChunk int    `json:"total_chunk" validate:"required"`
}

func (r *RequestHeaderDTO) Header(c *gin.Context) RequestHeaderDTO {
	if filename := c.Request.Header.Get("filename"); filename != "" {
		r.Filename = filename
	}

	if checkSum := c.Request.Header.Get("check-sum"); checkSum != "" {
		r.CheckSum = checkSum
	}

	if chunkIndex := c.Request.Header.Get("chunk-index"); chunkIndex != "" {
		if i, err := strconv.Atoi(chunkIndex); err == nil {
			r.ChunkIndex = i
		}
	}

	if totalChunk := c.Request.Header.Get("total-chunk"); totalChunk != "" {
		if i, err := strconv.Atoi(totalChunk); err == nil {
			r.TotalChunk = i
		}
	}

	return *r
}

type UploadChunkRequestServiceDTO struct {
	RequestHeader RequestHeaderDTO `json:"requestHeader" validate:"required"`
	Content       *bytes.Buffer    `json:"content" validate:"required"`
}
