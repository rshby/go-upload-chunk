package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/sirupsen/logrus"
	"go-upload-chunk/server/drivers/logger"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
)

func main() {
	logger.SetupLogger()

	var (
		filename   = "sample.jpeg"
		totalChunk = 100
		mu         = &sync.Mutex{}
	)

	// open file
	f, err := os.Open(fmt.Sprintf("./upload/%s", filename))
	if err != nil {
		logrus.Fatal(err)
	}

	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		logrus.Fatal(err)
	}

	// get file size
	fileSize := fileInfo.Size()
	logrus.Infof("file size : %d 🗂️", fileSize)
	chunkSize := fileSize / int64(totalChunk)

	// read and upload each chunk
	for i := 0; i < totalChunk; i++ {
		// lock mutex
		mu.Lock()

		var (
			start = int64(i) * chunkSize
			end   int64
		)

		if i == totalChunk-1 {
			// last chunk
			end = fileSize
		} else {
			end = start + chunkSize
		}

		// set offset to start value
		if _, err = f.Seek(start, io.SeekStart); err != nil {
			logrus.Errorf("Error seeking to chunk %d: %v", i, err)
			break
		}

		// read per chunk and save []byte to variable content
		content := make([]byte, end-start)
		if _, err = f.Read(content); err != nil {
			logrus.Error(err)
			break
		}

		// upload each chunk
		uploadChunk(filename, content, strconv.Itoa(i), strconv.Itoa(totalChunk))
		content = nil

		// unlock mutex
		mu.Unlock()
	}

	logrus.Infof("success upload")
}

// uploadChunk uploads file for each chunk
func uploadChunk(filename string, content []byte, chunkIndex, totalChunk string) {
	// create http client
	httpClient := &http.Client{}

	// create checksum
	hashChecksum := sha256.New()
	if _, err := io.Copy(hashChecksum, bytes.NewReader(content)); err != nil {
		logrus.Error(err)
		return
	}

	checksum := hex.EncodeToString(hashChecksum.Sum(nil))

	// create http request
	req, err := http.NewRequest(http.MethodPost, "http://localhost:4000/v1/file/chunk", bytes.NewReader(content))
	if err != nil {
		logrus.Error(err)
		return
	}

	// set header
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("filename", filename)
	req.Header.Add("check-sum", checksum)
	req.Header.Add("chunk-index", chunkIndex)
	req.Header.Add("total-chunk", totalChunk)

	// execute http call
	resp, err := httpClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return
	}

	defer resp.Body.Close()
}
