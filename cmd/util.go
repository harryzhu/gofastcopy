package cmd

import (
	"bufio"
	"encoding/hex"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeebo/xxh3"
)

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

func GetNowTime() time.Time {
	return time.Now()
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
	s = strings.ReplaceAll(s, ",", " ")
	layout := "2006-01-02 15:04:05"

	parsedTime, err := time.ParseInLocation(layout, s, time.Local)

	if err != nil {
		FatalError("TimeStr2Unix", err)
	}

	return parsedTime.Unix()
}

func Exists(fpath string) bool {
	_, err := os.Stat(fpath)
	if err != nil {
		return false
	}
	return true
}

func IsFileNeeded(fpath string, finfo fs.FileInfo) bool {
	if IsIgnoreDotFile == true {
		if strings.HasPrefix(filepath.Base(fpath), ".") {
			return false
		}
	}

	if FileExt != "" {
		if fextMatch.MatchString(filepath.Ext(fpath)) == false {
			return false
		}
	}

	fsize := finfo.Size()
	if MinSize != -1 {
		if fsize < MinSize {
			return false
		}
	}

	if MaxSize != -1 {
		if fsize > MaxSize {
			return false
		}
	}

	fmtime := finfo.ModTime().Unix()
	if MinAgeUnix != 0 {
		if fmtime < MinAgeUnix {
			return false
		}
	}

	if MaxAgeUnix != 0 {
		if fmtime > MaxAgeUnix {
			return false
		}
	}

	return true
}

func hashFile(fpath string) string {
	var hasher hash.Hash
	hasher = xxh3.New()

	fh, err := os.Open(fpath)
	if err != nil {
		PrintError("HashFile", err)
		return ""
	}

	r := bufio.NewReaderSize(fh, bufSize)

	var buf []byte = make([]byte, bufSize)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			PrintError("HashFile", err)
		}
		hasher.Write(buf[:n])
	}

	fh.Close()
	return hex.EncodeToString(hasher.Sum(nil))
}

func IsSame(srcPath string, dstPath string) bool {
	if Exists(dstPath) {
		var srcHash, dstHash string
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func(fpath string) {
			srcHash = hashFile(fpath)
			wg.Done()
		}(srcPath)
		go func(fpath string) {
			dstHash = hashFile(fpath)
			wg.Done()
		}(dstPath)
		wg.Wait()

		if srcHash == dstHash && srcHash != "" {
			return true
		}
	}

	return false
}

func showSpeed(dSec float64) error {
	sNum := atomic.LoadInt64(&srcNum)
	tSize := atomic.LoadInt64(&totalSize)
	tNum := atomic.LoadInt64(&totalNum)
	tSpeedMB := int64(0)
	if dSec > 0 {
		tSpeedMB = int64(float64(tSize)/dSec) >> 20
	}
	PrintlnInfo("green", "Stat", tSpeedMB, " MB/s, ", tNum, "/", sNum, " Files, ", tSize>>20, " MB (", tSize, " Bytes)")
	return nil
}
