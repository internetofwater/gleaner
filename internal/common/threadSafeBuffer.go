package common

import (
	"bytes"
	"sync"
)

type Buffer struct {
	b bytes.Buffer
	m sync.Mutex
}
