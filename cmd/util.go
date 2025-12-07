package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func FlagsValidate() error {
	if SourceDir == "" || TargetDir == "" || SourceDir == TargetDir {
		PrintError("gofastcopy", NewError("--source-dir=  --target-dir=  cannot be empty or same"))
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
	fmt.Println("threads: ", GetThreadNum())

	return nil
}

func SendFileToChanFile(srcPath string, dstPath string, info os.FileInfo, copyMode int) (ele map[string]any, err error) {
	srcPath = ToUnixSlash(srcPath)
	dstPath = ToUnixSlash(dstPath)

	ele = make(map[string]any)

	var fdata []byte
	if copyMode == 1 {
		fdata, err = os.ReadFile(srcPath)
		if err != nil {
			PrintError("SendFileToChanFile: os.ReadFile", err)
			return ele, err
		}
	}

	ele["srcPath"] = srcPath
	ele["dstPath"] = dstPath
	ele["FileInfo"] = info
	ele["FileData"] = fdata
	ele["CopyMode"] = copyMode

	return ele, nil
}

func GetChanFileToDisk(ele map[string]any) error {
	fsrc := ele["srcPath"].(string)
	fdst := ele["dstPath"].(string)
	finfo := ele["FileInfo"].(os.FileInfo)
	fdata := ele["FileData"].([]byte)
	fmode := ele["CopyMode"].(int)

	DebugInfo("GetChanFileToDisk: fsrc = ", fsrc, ", fdst = ", fdst)

	dstDir := filepath.Dir(fdst)
	//fmt.Println("dstDir:", dstDir)
	MakeDirs(dstDir)

	var err error
	if fmode == 1 {
		err = os.WriteFile(fdst, fdata, finfo.Mode())
		PrintError("GetChanFileToDisk", err)
		return err
	}

	if fmode == 0 || fdata == nil {
		err = CopyFile(fsrc, fdst)
		PrintError("GetChanFileToDisk", err)
		return err
	}

	return nil
}

func FastCopy() error {
	var num int = 0
	var numSkip map[string]int = make(map[string]int)
	numSkip["skip_dot_file"] = 0
	numSkip["skip_file_ext"] = 0
	numSkip["skip_size_min"] = 0
	numSkip["skip_size_max"] = 0
	numSkip["skip_age_min"] = 0
	numSkip["skip_age_max"] = 0
	numSkip["skip_exclude"] = 0

	qcap := GetThreadNum()

	var chanFile chan map[string]any = make(chan map[string]any, qcap)

	totalWriteSize := int64(0)
	totalSpeed := int64(0)
	//
	wg := sync.WaitGroup{}

	wg.Add(2)

	go func() {
		defer wg.Done()

		timeGetStart := GetNowUnix()
		timeGetStop := int64(0)

		wgGetChanFile := sync.WaitGroup{}
		numGet := int32(0)
		for {
			cf := <-chanFile
			if val, ok := cf["_COPYSTATUS"]; ok {
				DebugInfo("_COPYSTATUS:", val)
				break
			}

			totalWriteSize += cf["FileInfo"].(os.FileInfo).Size()

			atomic.AddInt32(&numGet, 1)
			wgGetChanFile.Add(1)

			go func(cf map[string]any) {
				defer func() {
					atomic.AddInt32(&numGet, -1)
					wgGetChanFile.Done()
				}()

				GetChanFileToDisk(cf)

			}(cf)

			curNumGet := atomic.LoadInt32(&numGet)

			if curNumGet > int32(qcap-1) && curNumGet%int32(qcap) == 0 {
				wgGetChanFile.Wait()
				timeGetStop = GetNowUnix()
				timeDuration := timeGetStop - timeGetStart
				if timeDuration > 0 {
					totalSpeed = totalWriteSize / timeDuration
				}
			}

		}
		wgGetChanFile.Wait()

	}()

	go func() {
		defer wg.Done()

		wgSendChanFile := sync.WaitGroup{}
		numSend := int32(0)

		filepath.Walk(SourceDir, func(fpath string, info os.FileInfo, err error) error {
			if err != nil {
				PrintError("FastCopy: walk", err)
				return err
			}

			fpath = ToUnixSlash(fpath)

			if info.IsDir() {
				if IsIgnoreEmptyFolder {
					return nil
				}

				D2Dir := strings.Replace(fpath, SourceDir, TargetDir, 1)
				_, err := os.Stat(D2Dir)
				if err != nil {
					MakeDirs(D2Dir)
					err = os.Chtimes(D2Dir, info.ModTime(), info.ModTime())
					PrintError("FastCopy", err)

					err = os.Chmod(D2Dir, info.Mode())
					PrintError("FastCopy", err)
					return err
				}

				return nil
			}

			num++
			if num < 500 || num%50 == 0 {
				fmt.Printf(" ...%3d, %10d, %20dMB/s\r", len(chanFile), num, totalSpeed>>20)
			}

			if FileExt != "" {
				fext := strings.ToLower(filepath.Ext(fpath))
				if fext != strings.ToLower(FileExt) {
					numSkip["skip_file_ext"] += 1
					return nil
				}
			}

			if IsIgnoreDotFile == true {
				if strings.HasPrefix(filepath.Base(fpath), ".") {
					numSkip["skip_dot_file"] += 1
					return nil
				}
			}

			if MinSize != -1 {
				if info.Size() < MinSize {
					numSkip["skip_size_min"] += 1
					return nil
				}
			}

			if MaxSize != -1 {
				if info.Size() > MaxSize {
					numSkip["skip_size_max"] += 1
					return nil
				}
			}

			if MinAge != "" {
				minAge := TimeStr2Unix(MinAge)
				if info.ModTime().Unix() < minAge {
					numSkip["skip_age_min"] += 1
					return nil
				}
			}

			if MaxAge != "" {
				maxAge := TimeStr2Unix(MaxAge)
				if info.ModTime().Unix() > maxAge {
					numSkip["skip_age_max"] += 1
					return nil
				}
			}

			if ExcludeDir != "" {
				excludePath := strings.Replace(fpath, SourceDir, ExcludeDir, 1)
				//DebugInfo("FastCopy: excludePath", excludePath)

				_, err = os.Stat(excludePath)
				if err == nil {
					numSkip["skip_exclude"] += 1
					DebugInfo("FastCopy: SKIP", excludePath)
					return nil
				}
			}

			targetPath := strings.Replace(fpath, SourceDir, TargetDir, 1)
			//DebugInfo("FastCopy: targetPath", targetPath)

			//
			_, err = os.Stat(fpath)
			if err != nil {
				return nil
			}

			if IsOverwrite == false {
				_, err = os.Stat(targetPath)
				if err == nil {
					return nil
				}
			}

			atomic.AddInt32(&numSend, 1)

			wgSendChanFile.Add(1)
			go func(fpath string, targetPath string, info os.FileInfo) error {
				defer func() {
					atomic.AddInt32(&numSend, -1)
					wgSendChanFile.Done()
				}()

				fmode := int(0)
				// 0: send path string
				// 1: cache file data in memory
				if IsWithLimitMemory == false {
					fmode = 1
				}

				if info.Size() > (16 << 20) {
					fmode = 0
				}

				ele, err := SendFileToChanFile(fpath, targetPath, info, fmode)
				if err != nil {
					return err
				}

				chanFile <- ele
				return nil

			}(fpath, targetPath, info)

			curNumSend := atomic.LoadInt32(&numSend)

			if curNumSend > int32(qcap-1) && curNumSend%int32(qcap) == 0 {
				wgSendChanFile.Wait()
			}

			return nil
		})

		copyDone := make(map[string]any)
		copyDone["_COPYSTATUS"] = "DONE"
		chanFile <- copyDone
		//
		wgSendChanFile.Wait()
	}()

	wg.Wait()

	fmt.Printf(" ... %20d\n", num)
	close(chanFile)

	for k, v := range numSkip {
		if v > 0 {
			fmt.Printf("\n** Ignored: %s: %v **", k, v)
		}

	}

	fmt.Printf("\n\n***** Total: %d *****\n", num)

	return nil
}

func CopyFile(src, dst string) error {
	src = ToUnixSlash(src)
	dst = ToUnixSlash(dst)
	srcFileHandler, err := os.Open(src)
	if err != nil {
		PrintError("CopyFile: os.Open", err)
		return err
	}
	defer srcFileHandler.Close()

	dstTemp := dst + ".ing"

	dstFileHandler, err := os.Create(dstTemp)
	if err != nil {
		PrintError("CopyFile: os.Create", err)
		return err
	}
	defer dstFileHandler.Close()

	srcReader := bufio.NewReader(srcFileHandler)
	dstWriter := bufio.NewWriter(dstFileHandler)
	_, err = io.Copy(dstWriter, srcReader)
	if err != nil {
		PrintError("CopyFile: io.Copy", err)
		return err
	}

	dstWriter.Flush()

	finfo, err := srcFileHandler.Stat()
	PrintError("CopyFile", err)

	err = os.Chtimes(dstTemp, finfo.ModTime(), finfo.ModTime())
	PrintError("CopyFile: os.Chtimes", err)

	srcFileHandler.Close()
	dstFileHandler.Close()

	err = os.Rename(dstTemp, dst)
	PrintError("CopyFile: os.Rename", err)

	err = os.Chmod(dst, finfo.Mode())
	PrintError("CopyFile: os.Chmod", err)

	return nil
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

func GetThreadNum() int {
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
	layout := "20060102150405"
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
