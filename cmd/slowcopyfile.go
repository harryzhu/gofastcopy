package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyFile(src, dst string, finfo os.FileInfo) (writeSize int64, err error) {
	if finfo.Size() > minDiffSize {
		if IsSame(src, dst) == true {
			return 0, nil
		}
	}
	MakeDirs(filepath.Dir(dst))

	if CopyMode == 0 {
		writeSize, err = zeroCopyFile(src, dst, finfo)
		if err == nil {
			return writeSize, nil
		} else {
			PrintError("zeroCopyFile", err)
		}
	}

	if CopyMode == 1 {
		writeSize, err = simdCopyFile(src, dst, finfo)
		if err == nil {
			return writeSize, err
		} else {
			PrintError("simdCopyFile", err)
		}
	}

	writeSize, err = slowCopyFile(src, dst, finfo)
	PrintError("slowCopyFile", err)

	return writeSize, err
}

func slowCopyFile(src, dst string, finfo os.FileInfo) (writeSize int64, err error) {
	srcFileHandler, err := os.Open(src)
	if err != nil {
		PrintError("CopyFile: os.Open", err)
		return 0, err
	}

	dstTemp := strings.Join([]string{dst, "ing"}, ".")
	dstFileHandler, err := os.Create(dstTemp)
	if err != nil {
		PrintError("CopyFile: os.Create", err)
		return 0, err
	}

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

	if err := chmodFile(dst, finfo); err != nil {
		return 0, err
	}

	return finfo.Size(), nil
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

func copySymlink(src, dst string) (n int, err error) {
	src = ToUnixSlash(src)
	dst = ToUnixSlash(dst)

	linfo, err := os.Lstat(src)
	if err != nil {
		PrintError("copyLink: os.Lstat", err)
		return 0, err
	}

	if Exists(dst) && IsOverwrite == false {
		return 1, nil
	}

	if Exists(dst) {
		err = os.Remove(dst)
		PrintError("copyLink: os.Remove", err)
	}

	if linfo.Mode()&os.ModeSymlink != 0 {
		DebugInfo("copyLink", strings.TrimLeft(src, SourceDir), ": is a symblink")
		srcLinkTarget, err := os.Readlink(src)
		if err != nil {
			PrintError("copyLink: os.Readlink", err)
			return 0, err
		}
		srcLinkTarget = strings.Replace(srcLinkTarget, SourceDir, TargetDir, 1)

		MakeDirs(filepath.Dir(dst))

		err = os.Symlink(srcLinkTarget, dst)
		if err != nil {
			PrintError("copyLink: os.Symlink", err)
			return 0, err
		}
		return 1, nil
	}
	return 0, NewError("ErrNotSymLink")
}

func zeroCopyFile(src, dst string, finfo os.FileInfo) (writeSize int64, err error) {
	srcFileHandler, err := os.Open(src)
	if err != nil {
		PrintError("zeroCopyFile: os.Open", err)
		return 0, err
	}

	dstTemp := strings.Join([]string{dst, "ing"}, ".")
	dstFileHandler, err := os.Create(dstTemp)
	if err != nil {
		PrintError("zeroCopyFile: os.Create", err)
		return 0, err
	}

	writeSize, err = dstFileHandler.ReadFrom(srcFileHandler)
	PrintError("zeroCopyFile: os.Open", err)

	srcFileHandler.Close()
	dstFileHandler.Close()

	err = os.Rename(dstTemp, dst)
	if err != nil {
		PrintError("zeroCopyFile: os.Rename", err)
		return 0, err
	}

	if err := chmodFile(dst, finfo); err != nil {
		return 0, err
	}

	return writeSize, nil
}
