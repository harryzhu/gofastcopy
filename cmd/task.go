package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

func taskSelectFiles() error {
	var dstPath string
	var excPath string
	var ele CopyElement
	filepath.Walk(SourceDir, func(fpath string, finfo fs.FileInfo, err error) error {
		if err != nil {
			PrintError("taskSelectFiles", err)
		}

		fpath = ToUnixSlash(fpath)
		dstPath = ToUnixSlash(strings.Replace(fpath, SourceDir, TargetDir, 1))

		if IsOverwrite == false {
			if Exists(dstPath) {
				DebugInfo("[SKIP]", dstPath)
				return nil
			}
		}

		if IsFollowSymlink == false {
			if isSymlink(fpath) {
				copySymlink(fpath, dstPath)
				return nil
			}
		}

		if finfo.IsDir() {
			if IsIgnoreEmptyFolder == false {
				MakeDirs(dstPath)
			}
			return nil
		}

		atomic.AddInt64(&srcNum, 1)

		if ExcludeDir != "" {
			excPath = ToUnixSlash(strings.Replace(fpath, SourceDir, ExcludeDir, 1))
			if Exists(excPath) {
				return nil
			}
		}

		if IsFileNeeded(fpath, finfo) == false {
			return nil
		}

		ele = CopyElement{
			Flag:  "",
			Src:   fpath,
			Dst:   dstPath,
			Finfo: finfo,
		}

		chanElement <- ele

		return nil
	})

	eleAllDone := CopyElement{
		Flag: FlagAllDone,
		Src:  "",
		Dst:  "",
	}

	chanElement <- eleAllDone

	return nil
}

func taskMirrorCleanTargetDir() error {
	PrintlnInfo("green", "taskMirrorClean", numTask)
	sem := make(chan struct{}, numTask)
	var srcPath string
	var rmErr error
	wg := sync.WaitGroup{}
	filepath.Walk(TargetDir, func(fpath string, finfo fs.FileInfo, err error) error {
		if err != nil {
			PrintError("taskMirrorCleanTargetDir", err)
		}

		fpath = ToUnixSlash(fpath)
		srcPath = ToUnixSlash(strings.Replace(fpath, TargetDir, SourceDir, 1))

		if finfo.IsDir() {
			return nil
		}

		if Exists(srcPath) == false {
			DebugInfo("Mirror: Remove", fpath)
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer func() {
					<-sem
					wg.Done()
				}()
				rmErr = os.Remove(fpath)
				PrintError("Mirror: Clean", rmErr)
			}()
		}
		return nil
	})

	wg.Wait()

	return nil
}

func taskFastCopy() error {
	PrintlnInfo("green", "taskFastCopy", numTask)
	sem := make(chan struct{}, numTask)
	var n int64
	var err error
	wg := sync.WaitGroup{}
	for {
		ch := <-chanElement
		if ch.Flag == FlagAllDone {
			break
		}
		if ch.Dst != "" && ch.Src != "" {
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer func() {
					<-sem
					wg.Done()
				}()
				n, err = copyFile(ch.Src, ch.Dst, ch.Finfo)
				if err != nil {
					PrintError("taskFastCopy", err)
				} else {
					atomic.AddInt64(&totalSize, n)
					atomic.AddInt64(&totalNum, 1)
				}
			}()
		}
	}

	wg.Wait()

	return nil
}
