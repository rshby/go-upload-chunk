package utils

import (
	"bytes"
	"sync"
)

var ByteBufferPool = &sync.Pool{New: func() any {
	b := &bytes.Buffer{}
	return b
}}
