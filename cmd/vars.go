package cmd

import (
	"errors"
	"io/fs"
	"regexp"
	"runtime"
)

const (
	MB int64 = 1 << 20
	//
	uMB uint64 = 1 << 20
	SEP string = "------------------------------------------------------------"
)

var (
	copymode map[int]string
)

var (
	// must be in multiples of 64
	bufSize int = 64 << 10
	Host    string
	Port    string
)

var (
	timeGetStart   int64                  = GetNowUnix()
	timeGetStop    int64                  = 0
	timeDuration   int64                  = 0
	totalWriteSize int64                  = 0
	totalSpeed     int64                  = 0
	totalNum       int                    = 0
	dirList        map[string]fs.DirEntry = make(map[string]fs.DirEntry, 2048)
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
	cpuFlags  string
)

var numStatistics map[string]int

var fextMatch *regexp.Regexp

var (
	ErrNotSymLink error = errors.New("invalid symlink")
)

var (
	copyAllDone CopyElement = CopyElement{Fsrc: "", Fdst: "", Finfo: nil, CopyMode: -1}
)
