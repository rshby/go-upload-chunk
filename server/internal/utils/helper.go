package utils

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"sync"
)

var ByteBufferPool = &sync.Pool{New: func() any {
	logrus.Info("create buffer from Pool")
	b := &bytes.Buffer{}
	return b
}}
