package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func bootstrap() error {
	qcap = getThreadNum()

	cpuFlags = getCPUFlags()

	numStatistics = make(map[string]int)
	numStatistics["skip_dot_file"] = 0
	numStatistics["skip_file_ext"] = 0
	numStatistics["skip_size_min"] = 0
	numStatistics["skip_size_max"] = 0
	numStatistics["skip_age_min"] = 0
	numStatistics["skip_age_max"] = 0
	numStatistics["skip_exclude_dir"] = 0
	numStatistics["skip_exists"] = 0
	//
	numStatistics["symbol_link"] = 0
	//
	copymode = make(map[int]string, 3)
	copymode[0] = "zero copy"
	copymode[1] = "simd copy"

	fextMatch = regexp.MustCompile("(?i)" + FileExt)

	return nil
}

func FormatPrint(ftype string, key string, args ...any) error {
	var s []string
	for _, arg := range args {
		s = append(s, fmt.Sprintf("%v", arg))
	}
	var f0 string
	switch {
	case ftype == "short":
		f0 = "%12s: %-20v\n"
	case ftype == "wide":
		f0 = "%25s: %-20v\n"
	default:
		f0 = "%12s: %-20v\n"

	}
	fmt.Printf(f0, key, strings.Join(s, " "))
	return nil
}

func argsValidate() error {
	if SourceDir == "" || TargetDir == "" || SourceDir == TargetDir {
		PrintError("gofastcopy", NewError("--source-dir=  --target-dir=  cannot be empty or same"))
		os.Exit(0)
	}

	if strings.HasPrefix(TargetDir, strings.TrimRight(SourceDir, "/")+"/") {
		PrintError("gofastcopy", NewError("--target-dir=  cannot be in --source-dir= "))
		ExitWithNum(0)
	}

	FormatPrint("short", "SourceDir", SourceDir)
	FormatPrint("short", "TargetDir", TargetDir)
	FormatPrint("short", "ExcludeDir", ExcludeDir)
	fmt.Println(SEP)
	//

	finfo, err := os.Stat(SourceDir)
	if err != nil {
		PrintError("gofastcopy", err)
		ExitWithNum(0)
	}

	if !finfo.IsDir() {
		PrintError("gofastcopy", NewError("--source-dir=", SourceDir, " should be a directory"))
		ExitWithNum(0)
	}

	//
	if TargetDir == "/" {
		PrintError("gofastcopy", NewError("--target-dir= cannot be \"/\" for safty"))
		ExitWithNum(0)
	}

	if strings.HasPrefix(strings.TrimRight(TargetDir, "/")+"/", strings.TrimRight(SourceDir, "/")+"/") {
		PrintError("gofastcopy", NewError("--target-dir= cannot be a subfolder in --source-dir= "))
		ExitWithNum(0)
	}

	//
	MakeDirs(TargetDir)
	finfo, err = os.Stat(TargetDir)
	if err != nil {
		PrintError("gofastcopy", err)
		ExitWithNum(0)
	}

	if !finfo.IsDir() {
		PrintError("gofastcopy", NewError("--target-dir=", TargetDir, " should be a directory"))
		ExitWithNum(0)
	}

	if FileExt != "" {
		FormatPrint("wide", "FileExtenion", FileExt)
	} else {
		FormatPrint("wide", "FileExtenion", "*")
	}

	var minAge, maxAge int64
	if MinAge != "" {
		minAge = TimeStr2Unix(MinAge)
		FormatPrint("wide", "latest-update-time: min", minAge)
	}

	if MaxAge != "" {
		maxAge = TimeStr2Unix(MaxAge)
		FormatPrint("wide", "latest-update-time: max", maxAge)
	}

	if minAge > 0 && maxAge > 0 && minAge > maxAge {
		PrintError("argsValidate", NewError("--min-age= cannot be greater than --max-age= "))
		ExitWithNum(0)
	}

	if MinSizeMB >= 0 {
		MinSize = MinSizeMB << 20
	}

	if MaxSizeMB >= 0 {
		MaxSize = MaxSizeMB << 20
	}

	if MinSize != -1 {
		FormatPrint("wide", "file-size: min", MinSize)
	}

	if MaxSize != -1 {
		FormatPrint("wide", "file-size: max", MaxSize)
	}

	if MinSize > -1 && MaxSize > -1 && MinSize > MaxSize {
		PrintError("argsValidate", NewError("--min-size= cannot be greater than --max-size= "))
		ExitWithNum(0)
	}

	if MinSize < -1 || MaxSize < -1 {
		PrintError("argsValidate", NewError("--min-size= or --max-size= should be greater than 0 "))
		ExitWithNum(0)
	}

	bootstrap()
	copymodestr := "slow copy"
	if _, ok := copymode[CopyMode]; ok {
		copymodestr = copymode[CopyMode]
	}

	FormatPrint("wide", "CopyElementWidth", unsafe.Sizeof(CopyElement{}))
	FormatPrint("wide", "ignore-dot-files", IsIgnoreDotFile)
	FormatPrint("wide", "ignore-empty-folder", IsIgnoreEmptyFolder)
	FormatPrint("wide", "overwrite-existing-files", IsOverwrite)
	FormatPrint("wide", "serial", IsSerial)
	FormatPrint("wide", "copy-mode", CopyMode, copymodestr)
	FormatPrint("wide", "purge", IsPurge)
	FormatPrint("wide", "cpu", numCPU, cpuFlags)
	FormatPrint("wide", "threads", qcap)
	FormatPrint("wide", "buffer", bufSize)
	FormatPrint("wide", "Time", time.Now().Format("2006-01-02 15:04:05"))
	return nil
}

func isSymlink(src string) bool {
	linfo, err := os.Lstat(src)
	if err != nil {
		PrintError("isSymblink", err)
		return false
	}
	if linfo.Mode()&os.ModeSymlink != 0 {
		return true
	}
	return false
}

func getThreadNum() int {
	if ThreadNum > 0 {
		return ThreadNum
	}
	qcap := numCPU * 2
	if qcap < 16 {
		qcap = 16
	}

	if qcap > 128 {
		qcap = 128
	}

	if IsWithLimitMemory {
		qcap = 4
	}

	return qcap
}

func MakeDirs(dpath string) error {
	dpath = ToUnixSlash(dpath)
	_, err := os.Stat(dpath)
	if err != nil {
		DebugInfo("MakeDirs", dpath)
		err = os.MkdirAll(dpath, os.ModePerm)
		PrintError("MakeDirs:MkdirAll", err)
		return err
	}
	return nil
}

func Int2Str(n int) string {
	return strconv.Itoa(n)
}

func Int64Int(n int64) int {
	n10, err := strconv.Atoi(strconv.FormatInt(n, 10))
	if err != nil {
		PrintError("Int64Int", err)
		return 0
	}
	return n10
}

func GetNowUnix() int64 {
	return time.Now().UTC().Unix()
}

func GetNowUnixMilli() int64 {
	return time.Now().UTC().UnixMilli()
}

func ToUnixSlash(s string) string {
	// for windows
	return strings.ReplaceAll(s, "\\", "/")
}

func TimeStr2Unix(s string) int64 {
	layout := "2006-01-02,15:04:05"
	var parsedTime time.Time
	var err error

	if IsWithTimeUTC {
		parsedTime, err = time.Parse(layout, s)
	} else {
		parsedTime, err = time.ParseInLocation(layout, s, time.Local)
	}

	if err != nil {
		PrintError("TimeStr2Unix", err)
		os.Exit(0)
	}

	return parsedTime.Unix()
}

func ExitWithNum(n int) {
	os.Exit(n)
}

func FileExists(fpath string) bool {
	_, err := os.Stat(fpath)
	if err != nil {
		return false
	}
	return true
}
