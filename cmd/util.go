package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	MB uint64 = 1 << 20
)

var (
	bufSize int = 64 << 10
	bufPool *BufferPool
)

func flagsValidate() error {
	if SourceDir == "" || TargetDir == "" || SourceDir == TargetDir {
		PrintError("gofastcopy", NewError("--source-dir=  --target-dir=  cannot be empty or same"))
		os.Exit(0)
	}

	if strings.HasPrefix(TargetDir, strings.TrimRight(SourceDir, "/")+"/") {
		PrintError("gofastcopy", NewError("--target-dir=  cannot be in --source-dir= "))
		os.Exit(0)
	}

	fmt.Printf("SourceDir: %v\n", SourceDir)
	fmt.Printf("TargetDir: %v\n", TargetDir)
	fmt.Printf("ExcludeDir: %v\n", ExcludeDir)
	//

	finfo, err := os.Stat(SourceDir)
	if err != nil {
		PrintError("gofastcopy", err)
		os.Exit(0)
	}
	if !finfo.IsDir() {
		PrintError("gofastcopy", NewError("--source-dir=", SourceDir, " should be a directory"))
		os.Exit(0)
	}
	//
	if TargetDir == "/" {
		PrintError("gofastcopy", NewError("--target-dir= cannot be \"/\" for safty"))
		os.Exit(0)
	}

	if strings.HasPrefix(strings.TrimRight(TargetDir, "/")+"/", strings.TrimRight(SourceDir, "/")+"/") {
		PrintError("gofastcopy", NewError("--target-dir= cannot be a subfolder in --source-dir= "))
		os.Exit(0)
	}

	//
	MakeDirs(TargetDir)
	finfo, err = os.Stat(TargetDir)
	if err != nil {
		PrintError("gofastcopy", err)
		os.Exit(0)
	}
	if !finfo.IsDir() {
		PrintError("gofastcopy", NewError("--target-dir=", TargetDir, " should be a directory"))
		os.Exit(0)
	}

	if FileExt != "" {
		fmt.Println("FileExtenion: ", FileExt)
	} else {
		fmt.Println("FileExtenion: *")
	}

	var minAge, maxAge int64
	if MinAge != "" {
		minAge = TimeStr2Unix(MinAge)
		fmt.Println("latest update time: min: ", minAge)
	}

	if MaxAge != "" {
		maxAge = TimeStr2Unix(MaxAge)
		fmt.Println("latest update time: max: ", maxAge)
	}

	if minAge > 0 && maxAge > 0 && minAge > maxAge {
		PrintError("FlagsValidate", NewError("--min-age= cannot be greater than --max-age= "))
		os.Exit(0)
	}

	if MinSizeMB >= 0 {
		MinSize = MinSizeMB << 20
	}

	if MaxSizeMB >= 0 {
		MaxSize = MaxSizeMB << 20
	}

	if MinSize != -1 {
		fmt.Println("file size: min: ", MinSize)
	}

	if MaxSize != -1 {
		fmt.Println("file size: max: ", MaxSize)
	}

	if MinSize > -1 && MaxSize > -1 && MinSize > MaxSize {
		PrintError("FlagsValidate", NewError("--min-size= cannot be greater than --max-size= "))
		os.Exit(0)
	}

	if MinSize < -1 || MaxSize < -1 {
		PrintError("FlagsValidate", NewError("--min-size= or --max-size= should be greater than 0 "))
		os.Exit(0)
	}

	cpuflags := getCPUFlags()
	if IsSIMD {
		if cpuflags == "" {
			IsSIMD = false
		}
	}

	if IsSerial || IsWithLimitMemory {
		bufSize = 32 << 10
	}

	bufPool = NewBufferPool(bufSize, 4096)

	fmt.Println("ignore dot files: ", IsIgnoreDotFile)
	fmt.Println("ignore empty folder: ", IsIgnoreEmptyFolder)
	fmt.Println("overwrite existing files: ", IsOverwrite)
	fmt.Println("serial: ", IsSerial)
	fmt.Println("simd: ", IsSIMD)
	fmt.Println("purge: ", IsPurge)
	fmt.Println("cpu: ", numCPU, cpuflags)
	fmt.Println("threads: ", getThreadNum())
	fmt.Println("buffer: ", bufSize)
	fmt.Println("Time: ", time.Now().Format("2006-01-02 15:04:05"))

	return nil
}

func copyFile(src, dst string, finfo os.FileInfo) (writeSize int64, err error) {
	if IsSIMD {
		writeSize, err = simdCopyFile(src, dst, finfo)
		if err == nil {
			return writeSize, err
		}
	}

	writeSize, err = regularCopyFile(src, dst, finfo)

	return writeSize, err
}

func regularCopyFile(src, dst string, finfo os.FileInfo) (writeSize int64, err error) {
	srcFileHandler, err := os.Open(src)
	if err != nil {
		PrintError("CopyFile: os.Open", err)
		return 0, err
	}
	defer srcFileHandler.Close()

	dstTemp := strings.Join([]string{dst, "ing"}, ".")

	MakeDirs(filepath.Dir(dstTemp))

	dstFileHandler, err := os.Create(dstTemp)
	if err != nil {
		PrintError("CopyFile: os.Create", err)
		return 0, err
	}
	defer dstFileHandler.Close()

	buf := make([]byte, bufSize)
	_, err = io.CopyBuffer(dstFileHandler, srcFileHandler, buf)
	if err != nil {
		PrintError("CopyFile: io.CopyBuffer", err)
		return 0, err
	}

	srcFileHandler.Close()
	dstFileHandler.Close()

	err = os.Rename(dstTemp, dst)
	if err != nil {
		PrintError("CopyFile: os.Rename", err)
		return 0, err
	}

	err = os.Chmod(dst, finfo.Mode())
	if err != nil {
		PrintError("CopyFile: os.Chmod", err)
		return 0, err
	}

	err = os.Chtimes(dst, finfo.ModTime(), finfo.ModTime())
	if err != nil {
		PrintError("CopyFile: os.Chtimes", err)
		return 0, err
	}

	return finfo.Size(), nil
}

func copyLink(src, dst string) error {
	src = ToUnixSlash(src)
	dst = ToUnixSlash(dst)

	if _, err := os.Stat(dst); err == nil {
		return nil
	}

	linfo, err := os.Lstat(src)
	if err != nil {
		PrintError("copyLink", err)
		return err
	}

	if linfo.Mode()&os.ModeSymlink != 0 {
		DebugInfo("copyLink", strings.TrimLeft(src, SourceDir), ": is a symblink")
		srcLinkTarget, err := os.Readlink(src)
		if err != nil {
			PrintError("copyLink", err)
			return err
		}
		DebugInfo("copyLink", src, " -> ", srcLinkTarget)

		MakeDirs(filepath.Dir(dst))

		err = os.Symlink(srcLinkTarget, dst)
		PrintError("copyLink: Symlink", err)

	}
	return nil
}

func isSymblink(src string) bool {
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
	qcap := numCPU * 5
	if qcap < 32 {
		qcap = 32
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
