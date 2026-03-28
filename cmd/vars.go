package cmd

import (
	"os"
	"runtime"
)

const (
	MB uint64 = 1 << 20
)

var (
	// if change bufSize, then [uintptrAlign] has to be changed as well
	bufSize int = 64 << 10
)

var (
	isAVX512       bool
	isAVX2         bool
	isSSE3         bool
	isASIMD        bool
	uintptrAlign   uintptr = uintptr(64)
	uintptrBufSize uintptr = uintptr(bufSize)
)

var (
	timeGetStart   int64                  = GetNowUnix()
	timeGetStop    int64                  = 0
	timeDuration   int64                  = 0
	totalWriteSize int64                  = 0
	totalSpeed     int64                  = 0
	dirList        map[string]os.FileInfo = make(map[string]os.FileInfo, 1024)
)

var (
	qcap             int
	chanFile         chan CopyElement = make(chan CopyElement, 4096)
	isChanFileRWDone bool
	taskNumGet       int32 = 0
)

var (
	memStats  runtime.MemStats
	memString string
)
