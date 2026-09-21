package cmd

import (
	"io/fs"
	"regexp"
)

type CopyElement struct {
	Flag  string
	Src   string
	Dst   string
	Finfo fs.FileInfo
}

var (
	chanElement chan CopyElement = make(chan CopyElement, 4096)
)

var (
	FlagAllDone string = "__ALL_DONE__"
	minDiffSize int64  = 128 << 20
	isOnWindows bool   = false
)

var (
	IsDebug             bool
	IsIgnoreDotFile     bool
	IsIgnoreEmptyFolder bool
	IsFollowSymlink     bool
	IsOverwrite         bool
	IsMirror            bool
	MaxSize             int64
	MinSize             int64
	MaxSizeMB           int64
	MinSizeMB           int64
	MinAge              string
	MaxAge              string
	SourceDir           string
	TargetDir           string
	ExcludeDir          string
	FileExt             string
	//
	IsSerial bool
	CopyMode int
)

var (
	MinAgeUnix int64
	MaxAgeUnix int64
	fextMatch  *regexp.Regexp
)

var (
	timeStart  int64
	timeStop   int64
	numCPU     int = 4
	numTask    int = 4
	bufSize    int = 64 << 10
	totalSize  int64
	totalNum   int64
	totalSpeed int64
	srcNum     int64
)
