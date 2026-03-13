package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

	fmt.Println("ignore dot files: ", IsIgnoreDotFile)
	fmt.Println("ignore empty folder: ", IsIgnoreEmptyFolder)
	fmt.Println("overwrite existing files: ", IsOverwrite)
	fmt.Println("serial: ", IsSerial)
	fmt.Println("purge: ", IsPurge)
	fmt.Println("threads: ", getThreadNum())
	fmt.Println("Time: ", time.Now().Format("2006-01-02 15:04:05"))

	return nil
}

func copyFile(src, dst string) (writeSize int64, err error) {
	src = ToUnixSlash(src)
	dst = ToUnixSlash(dst)
	srcFileHandler, err := os.Open(src)
	if err != nil {
		PrintError("CopyFile: os.Open", err)
		return 0, err
	}
	defer srcFileHandler.Close()

	dstTemp := dst + ".ing"

	MakeDirs(filepath.Dir(dstTemp))

	dstFileHandler, err := os.Create(dstTemp)
	if err != nil {
		PrintError("CopyFile: os.Create", err)
		return 0, err
	}
	defer dstFileHandler.Close()

	//
	srcReader := bufio.NewReader(srcFileHandler)
	dstWriter := bufio.NewWriter(dstFileHandler)

	_, err = io.Copy(dstWriter, srcReader)
	if err != nil {
		PrintError("CopyFile: io.Copy", err)
		return 0, err
	}

	dstWriter.Flush()

	finfo, err := srcFileHandler.Stat()
	if err != nil {
		PrintError("CopyFile", err)
		return 0, err
	}

	err = os.Chtimes(dstTemp, finfo.ModTime(), finfo.ModTime())
	if err != nil {
		PrintError("CopyFile: os.Chtimes", err)
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

	return finfo.Size(), nil
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
