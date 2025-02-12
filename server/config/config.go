package config

import (
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

// GetEnv retrieves value from env
func GetEnv(k string) string {
	if err := godotenv.Load(".env"); err != nil {
		logrus.Fatalf("failed load env : %s", err.Error())
	}

	return os.Getenv(k)
}

// Mode retrieves mode from env
func Mode() string {
	return GetEnv("MODE")
}

// Port retrieves port value from env
func Port() int {
	if val := GetEnv("PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			return port
		}
	}

	// default port
	return 4000
}

// FolderUploadChunk retrieves path folder to save chunk folder
func FolderUploadChunk() string {
	return GetEnv("FOLDER_UPLOAD_CHUNK")
}
