package cmd

import (
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/cpu"
)

func simdCopyFile(src, dst string, finfo os.FileInfo) (writeSize int64, err error) {
	srcFd, err := syscall.Open(src, syscall.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}

	defer syscall.Close(srcFd)

	dstFd, err := syscall.Open(dst, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}
	defer syscall.Close(dstFd)

	//alignedBuf := make([]byte, bufSize)
	alignedBuf := bufPool.Get()
	for {
		n, err := syscall.Read(srcFd, alignedBuf)
		if n == 0 || err != nil {
			break
		}

		simdCopy(alignedBuf[:n])

		_, err = syscall.Write(dstFd, alignedBuf[:n])
		if err != nil {
			return 0, err
		}
	}

	bufPool.Put(alignedBuf)

	err = os.Chmod(dst, finfo.Mode())
	PrintError("copyFileSIMD: Chmod", err)

	err = os.Chtimes(dst, finfo.ModTime(), finfo.ModTime())
	PrintError("copyFileSIMD: Chtimes", err)

	return finfo.Size(), nil
}

func simdCopy(data []byte) {
	switch {
	case cpu.X86.HasAVX2:
		avx2Copy(data)
	case cpu.X86.HasSSE3:
		sseCopy(data)
	case cpu.ARM64.HasASIMD:
		neonCopy(data)
	}
}

func avx2Copy(data []byte) {
}
func sseCopy(data []byte) {
}
func neonCopy(data []byte) {
}

func getCPUFlags() string {
	cfs := []string{}
	if cpu.X86.HasAVX2 {
		cfs = append(cfs, "avx2")
	}

	if cpu.X86.HasSSE3 {
		cfs = append(cfs, "sse3")
	}

	if cpu.ARM64.HasASIMD {
		cfs = append(cfs, "asimd")
	}

	if len(cfs) == 0 {
		return ""
	}

	return strings.Join(cfs, " ")
}
