package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	timeGetStart   int64                  = GetNowUnix()
	timeGetStop    int64                  = 0
	timeDuration   int64                  = 0
	totalWriteSize int64                  = 0
	totalSpeed     int64                  = 0
	dirList        map[string]os.FileInfo = make(map[string]os.FileInfo, 4096)
	memStats       runtime.MemStats
	memString      string
)

type CopyElement struct {
	Fsrc     string
	Fdst     string
	Finfo    os.FileInfo
	CopyMode int
}

func updateTotalSpeed() {
	timeGetStop = GetNowUnix()
	timeDuration = timeGetStop - timeGetStart
	if timeDuration > 0 {
		totalSpeed = totalWriteSize / timeDuration
	}

	if IsWithMemStats {
		runtime.ReadMemStats(&memStats)
		memString = fmt.Sprintf("MEM: %vMB,%vMB,NumGC: %v", memStats.Alloc/MB, memStats.Sys/MB, memStats.NumGC)
	}
}

func sendFileToChanFile(srcPath string, dstPath string, finfo os.FileInfo, copyMode int) (ele CopyElement, err error) {
	ele.Fsrc = ToUnixSlash(srcPath)
	ele.Fdst = ToUnixSlash(dstPath)
	ele.Finfo = finfo
	ele.CopyMode = copyMode

	return ele, nil
}

func getChanFileToDisk(ele CopyElement) error {
	if ele.Finfo.IsDir() {
		return nil
	}

	fsrc := ele.Fsrc
	fdst := ele.Fdst
	finfo := ele.Finfo

	DebugInfo("GetChanFileToDisk:", "fsrc = ", fsrc, ", fdst = ", fdst)

	_, err := copyFile(fsrc, fdst, finfo)

	PrintError("GetChanFileToDisk", err)
	return err
}

func fastCopy() error {
	var num int = 0
	var numSkip map[string]int = make(map[string]int)
	numSkip["skip_dot_file"] = 0
	numSkip["skip_file_ext"] = 0
	numSkip["skip_size_min"] = 0
	numSkip["skip_size_max"] = 0
	numSkip["skip_age_min"] = 0
	numSkip["skip_age_max"] = 0
	numSkip["skip_exclude_dir"] = 0
	numSkip["skip_exists"] = 0

	qcap := getThreadNum()

	var chanFile chan CopyElement = make(chan CopyElement, qcap)
	//
	var IsAllRWDone bool
	//
	wg := sync.WaitGroup{}

	wg.Add(3)

	go func() error {
		defer wg.Done()

		for {
			if IsAllRWDone == true {
				break
			}

			if num > 99 && num%100 == 0 {
				updateTotalSpeed()
				if IsSerial {
					fmt.Printf(" %s %10d, %20dMB/s\r", ":::", num, totalSpeed>>20)
				} else {
					fmt.Printf(" %s %10d, %10d, %25dMB/s,  %s\r", ":::", len(chanFile), num, totalSpeed>>20, memString)
				}

			}
		}
		return nil
	}()

	//chanFile
	go func() error {
		defer wg.Done()

		wgGetChanFile := sync.WaitGroup{}
		numGet := int32(0)
		qcapInt32 := int32(qcap)

		for {
			cf := <-chanFile
			if cf.CopyMode == -1 {
				IsAllRWDone = true
				DebugInfo("_COPYSTATUS:", "DONE")
				break
			}

			totalWriteSize += cf.Finfo.Size()

			atomic.AddInt32(&numGet, 1)
			wgGetChanFile.Add(1)

			go func(cf CopyElement) {
				defer func() {
					atomic.AddInt32(&numGet, -1)
					wgGetChanFile.Done()
				}()
				getChanFileToDisk(cf)
			}(cf)

			curNumGet := atomic.LoadInt32(&numGet)

			if curNumGet > qcapInt32 && curNumGet%qcapInt32 == 0 {
				wgGetChanFile.Wait()
			}

		}
		wgGetChanFile.Wait()

		updateTotalSpeed()
		return nil
	}()

	go func() error {
		defer wg.Done()

		fextreg := regexp.MustCompile("(?i)" + FileExt)
		var targetPath string
		var fsize int64

		filepath.Walk(SourceDir, func(fpath string, finfo os.FileInfo, err error) error {
			if err != nil {
				PrintError("FastCopy: walk", err)
				return err
			}

			fpath = ToUnixSlash(fpath)
			targetPath = ToUnixSlash(strings.Replace(fpath, SourceDir, TargetDir, 1))

			if finfo.IsDir() {
				if IsIgnoreEmptyFolder {
					return nil
				}

				D2Dir := strings.Replace(fpath, SourceDir, TargetDir, 1)
				if _, err := os.Stat(D2Dir); err != nil {
					MakeDirs(D2Dir)
				}
				dirList[targetPath] = finfo
				return nil
			}

			num++

			if isSymblink(fpath) {
				copyLink(fpath, targetPath)
				return nil
			}

			if FileExt != "" {
				if fextreg.MatchString(filepath.Ext(fpath)) == false {
					numSkip["skip_file_ext"]++
					return nil
				}
			}

			if IsIgnoreDotFile == true {
				if strings.HasPrefix(filepath.Base(fpath), ".") {
					numSkip["skip_dot_file"]++
					return nil
				}
			}

			fsize = finfo.Size()

			if MinSize != -1 {
				if fsize < MinSize {
					numSkip["skip_size_min"]++
					return nil
				}
			}

			if MaxSize != -1 {
				if fsize > MaxSize {
					numSkip["skip_size_max"]++
					return nil
				}
			}

			if MinAge != "" {
				minAge := TimeStr2Unix(MinAge)
				if finfo.ModTime().Unix() < minAge {
					numSkip["skip_age_min"]++
					return nil
				}
			}

			if MaxAge != "" {
				maxAge := TimeStr2Unix(MaxAge)
				if finfo.ModTime().Unix() > maxAge {
					numSkip["skip_age_max"]++
					return nil
				}
			}

			if ExcludeDir != "" {
				excludePath := strings.Replace(fpath, SourceDir, ExcludeDir, 1)
				//DebugInfo("FastCopy: excludePath", excludePath)

				_, err = os.Stat(excludePath)
				if err == nil {
					numSkip["skip_exclude_dir"]++
					DebugInfo("FastCopy: SKIP", excludePath)
					return nil
				}
			}

			if IsOverwrite == false {
				_, err = os.Stat(targetPath)
				if err == nil {
					numSkip["skip_exists"]++
					return nil
				}
			}

			//

			if IsDryRun {
				fmt.Printf("%s ==> %s\n", fpath, targetPath)
				return nil
			}

			if IsSerial {
				_, err = copyFile(fpath, targetPath, finfo)

				if err != nil {
					PrintError("FastCopy: copyFile", err)
					return err
				}

				totalWriteSize += fsize
				return nil
			}

			// fmode := int(0)
			// 0: send path string
			// 1: cache file data in memory
			// -1: _COPYSTATUS = Done

			ele, err := sendFileToChanFile(fpath, targetPath, finfo, 0)
			if err != nil {
				PrintError("FastCopy: sendFileToChanFile", err)
				return err
			}

			chanFile <- ele

			return nil
		})

		if IsSerial {
			fmt.Printf(" %s %10d\r", ":::", num)
		} else {
			fmt.Printf(" %s %10d, %10d, %25dMB/s\r", ":::", len(chanFile), num, totalSpeed>>20)
		}

		//
		copyAllDone := CopyElement{}

		copyAllDone.Fsrc = ""
		copyAllDone.Fdst = ""
		copyAllDone.CopyMode = -1
		// CopyMode = -1 means COPY STATUS = Done
		chanFile <- copyAllDone

		return nil
	}()

	wg.Wait()

	close(chanFile)

	fmt.Println("------------------------------------------------------------")
	var allIgnoredFiles int
	for k, v := range numSkip {
		fmt.Printf("\n** Ignored: %20s: %10v", k, v)
		allIgnoredFiles += v
	}

	if IsSerial {
		fmt.Printf("\n\n** Files: Total: %d, Copied: %d **\n", num, (num - allIgnoredFiles))
	} else {
		fmt.Printf("\n\n** Files: Total: %d, Copied: %d, Write: %d MB, Speed: %d MB/s **\n", num, (num - allIgnoredFiles), totalWriteSize>>20, totalSpeed>>20)
	}
	return nil
}

func purgeTargetDir() error {
	var e1, e2 error
	SourceDir = ToUnixSlash(SourceDir)
	TargetDir = ToUnixSlash(TargetDir)
	filepath.WalkDir(TargetDir, func(dstPath string, finfo fs.DirEntry, err error) error {
		if err != nil {
			PrintError("purgeTargetDir: walkdir", err)
			return err
		}

		dstPath = ToUnixSlash(dstPath)

		srcPath := strings.Replace(dstPath, strings.TrimRight(TargetDir, "/"), strings.TrimRight(SourceDir, "/"), 1)
		if _, e1 = os.Stat(srcPath); e1 != nil {
			e2 = os.Remove(dstPath)
			PrintError("purgeTargetDir:os.Remove", e2)
		}

		return nil
	})
	return nil
}

func updateTargetDir() error {
	if len(dirList) > 0 {
		var err error
		t1 := GetNowUnix()
		for dstPath, srcInfo := range dirList {
			err = os.Chtimes(dstPath, srcInfo.ModTime(), srcInfo.ModTime())
			PrintError("updateTargetDir: os.Chtimes", err)

			err = os.Chmod(dstPath, srcInfo.Mode())
			PrintError("updateTargetDir: os.Chmod", err)
		}
		t2 := GetNowUnix()

		DebugInfo("updateTargetDir", (t2 - t1), " (sec)/", len(dirList))
	}
	return nil
}
