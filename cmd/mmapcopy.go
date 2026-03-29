package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/mmap"
)

// do not use
func mmapCopyFile(src, dst string, finfo os.FileInfo) (writeSize int64, err error) {
	srcFileHandler, err := mmap.Open(src)
	if err != nil {
		PrintError("mmapCopyFile: mmap.Open", err)
		return 0, err
	}
	defer srcFileHandler.Close()

	data := make([]byte, srcFileHandler.Len())
	_, err = srcFileHandler.ReadAt(data, 0)

	MakeDirs(filepath.Dir(dst))

	dstTemp := strings.Join([]string{dst, "ing"}, ".")
	dstFileHandler, err := os.Create(dstTemp)
	if err != nil {
		PrintError("mmapCopyFile: os.Create", err)
		return 0, err
	}
	defer dstFileHandler.Close()

	dstWriter := bufio.NewWriterSize(dstFileHandler, bufSize)
	defer dstWriter.Flush()

	wlen := bufSize
	for len(data) > 0 {
		if wlen > len(data) {
			wlen = len(data)
		}
		_, err := dstWriter.Write(data[:wlen])
		if err != nil {
			PrintError("mmapCopyFile:dstWriter.Write", err)
			return 0, err
		}
		data = data[wlen:]
	}

	dstWriter.Flush()
	// err = os.WriteFile(dstTemp, data, finfo.Mode())
	// PrintError("mmapCopyFile: os.WriteFile", err)

	srcFileHandler.Close()
	dstFileHandler.Close()

	err = os.Rename(dstTemp, dst)
	if err != nil {
		PrintError("mmapCopyFile: os.Rename", err)
		return 0, err
	}

	err = os.Chmod(dst, finfo.Mode())
	if err != nil {
		PrintError("mmapCopyFile: os.Chmod", err)
		return 0, err
	}

	err = os.Chtimes(dst, finfo.ModTime(), finfo.ModTime())
	if err != nil {
		PrintError("mmapCopyFile: os.Chtimes", err)
		return 0, err
	}

	return finfo.Size(), nil
}
