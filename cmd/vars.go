package cmd

import (
	"os"
	"runtime"
)

const (
	MB int64 = 1 << 20
	//
	uMB uint64 = 1 << 20
)

var (
	bufSize int = 64 << 10
)

var (
	timeGetStart   int64                  = GetNowUnix()
	timeGetStop    int64                  = 0
	timeDuration   int64                  = 0
	totalWriteSize int64                  = 0
	totalSpeed     int64                  = 0
	dirList        map[string]os.FileInfo = make(map[string]os.FileInfo, 2048)
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
