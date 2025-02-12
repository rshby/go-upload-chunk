package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/sirupsen/logrus"
	"go-upload-chunk/server/drivers/logger"
	"io"
	"net/http"
	"os"
	"strconv"
)

func main() {
	logger.SetupLogger()

	filename := "tujuan_paket.pdf"
	totalChunk := 5

	// open file
	f, err := os.Open(fmt.Sprintf("./upload/%s", filename))
	if err != nil {
		logrus.Fatal(err)
	}

	defer f.Close()

	// create checksum final file
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		logrus.Fatal(err)
	}
	checksum := fmt.Sprintf("%x", h.Sum(nil))

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
		start := int64(i) * chunkSize
		end := start + chunkSize
		if i == totalChunk-1 {
			end = fileSize
		}

		_, _ = f.Seek(start, io.SeekStart)

		// read chunk
		buf := make([]byte, end-start)
		if _, err = f.Read(buf); err != nil {
			logrus.Error(err)
			break
		}

		// upload each chunk
		uploadChunk(filename, buf, checksum, strconv.Itoa(i), strconv.Itoa(totalChunk))
	}

	logrus.Infof("success upload")
}

func uploadChunk(filename string, content []byte, checksum, chunkIndex, totalChunk string) {
	// create http client
	httpClient := &http.Client{}

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

	logrus.Infof("check sum : %s", checksum)

	// execute http call
	resp, err := httpClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return
	}

	defer resp.Body.Close()
}
