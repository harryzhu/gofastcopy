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

func file2CopyElement(srcPath string, dstPath string, finfo os.FileInfo, copyMode int) (ele CopyElement, err error) {
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
	var numStatistics map[string]int = make(map[string]int)
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

	wg := sync.WaitGroup{}

	wg.Add(3)

	go func() error {
		defer wg.Done()
		for {
			if isChanFileRWDone == true {
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
		taskChanFile()
		updateTotalSpeed()
		return nil
	}()

	go func() error {
		defer wg.Done()

		fextreg := regexp.MustCompile("(?i)" + FileExt)
		var targetPath string
		var fsize int64
		var ftime int64

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
				MakeDirs(D2Dir)

				dirList[targetPath] = finfo
				return nil
			}

			num++

			if isSymblink(fpath) {

				nl, err := copyLink(fpath, targetPath)
				if err == nil {
					if nl == 0 {
						numStatistics["skip_exists"]++
					} else {
						numStatistics["symbol_link"] += 1
					}
				}
				return nil
			}

			if FileExt != "" {
				if fextreg.MatchString(filepath.Ext(fpath)) == false {
					numStatistics["skip_file_ext"]++
					return nil
				}
			}

			if IsIgnoreDotFile == true {
				if strings.HasPrefix(filepath.Base(fpath), ".") {
					numStatistics["skip_dot_file"]++
					return nil
				}
			}

			if MinSize != -1 || MaxSize != -1 {
				fsize = finfo.Size()
				if MinSize != -1 {
					if fsize < MinSize {
						numStatistics["skip_size_min"]++
						return nil
					}
				}

				if MaxSize != -1 {
					if fsize > MaxSize {
						numStatistics["skip_size_max"]++
						return nil
					}
				}
			}

			if MinAge != "" || MaxAge != "" {
				ftime = finfo.ModTime().Unix()
				if MinAge != "" {
					minAge := TimeStr2Unix(MinAge)
					if ftime < minAge {
						numStatistics["skip_age_min"]++
						return nil
					}
				}

				if MaxAge != "" {
					maxAge := TimeStr2Unix(MaxAge)
					if ftime > maxAge {
						numStatistics["skip_age_max"]++
						return nil
					}
				}
			}

			if ExcludeDir != "" {
				excludePath := strings.Replace(fpath, SourceDir, ExcludeDir, 1)
				//DebugInfo("FastCopy: excludePath", excludePath)

				_, err = os.Stat(excludePath)
				if err == nil {
					numStatistics["skip_exclude_dir"]++
					DebugInfo("FastCopy: SKIP", excludePath)
					return nil
				}
			}

			if IsOverwrite == false {
				_, err = os.Stat(targetPath)
				if err == nil {
					numStatistics["skip_exists"]++
					return nil
				}
			}

			//
			if IsSerial {
				_, err = copyFile(fpath, targetPath, finfo)
				if err != nil {
					PrintError("FastCopy: copyFile", err)
					return err
				}

				atomic.AddInt64(&totalWriteSize, finfo.Size())
				return nil
			}

			// fmode := int(0)
			// 0: send path string
			// 1: cache file data in memory
			// -1: _COPYSTATUS = Done

			ele, err := file2CopyElement(fpath, targetPath, finfo, 0)
			if err != nil {
				PrintError("FastCopy: file2CopyElement", err)
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
	for k, v := range numStatistics {
		if strings.HasPrefix(k, "skip_") {
			fmt.Printf("\n** Ignored: %20s: %10v", k, v)
			allIgnoredFiles += v
		}
	}

	fmt.Println("")

	for k, v := range numStatistics {
		if strings.HasPrefix(k, "skip_") == false {
			fmt.Printf("\n** Copied: %20s: %10v", k, v)
		}
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

func taskChanFile() error {
	wgGetChanFile := sync.WaitGroup{}
	numWait := int32(qcap)
	curNumGet := int32(0)

	for {
		cf := <-chanFile
		if cf.CopyMode == -1 {
			isChanFileRWDone = true
			DebugInfo("_COPYSTATUS:chanFile:", "DONE")
			break
		}

		atomic.AddInt64(&totalWriteSize, cf.Finfo.Size())

		atomic.AddInt32(&taskNumGet, 1)
		wgGetChanFile.Add(1)

		go func(cf CopyElement) {
			defer func() {
				atomic.AddInt32(&taskNumGet, -1)
				wgGetChanFile.Done()
			}()
			getChanFileToDisk(cf)
		}(cf)

		curNumGet = atomic.LoadInt32(&taskNumGet)

		if curNumGet%numWait == 0 {
			wgGetChanFile.Wait()
		}

	}
	wgGetChanFile.Wait()

	return nil
}
