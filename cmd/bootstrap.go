package cmd

import (
	"regexp"
	"runtime"
	"strings"
)

func Bootstrap() error {
	numCPU = runtime.NumCPU()
	numTask = max(numTask, numCPU)
	if IsSerial {
		numTask = 1
		minDiffSize = 32 << 20
	}

	if numTask > 64 {
		numTask = 64
	}

	if runtime.GOOS == "windows" {
		IsFollowSymlink = true
		PrintlnInfo("green", "--follow-symlink=true", "always \"true\" on Windows")
	}

	SourceDir = strings.TrimRight(ToUnixSlash(SourceDir), "/")
	TargetDir = strings.TrimRight(ToUnixSlash(TargetDir), "/")
	ExcludeDir = strings.TrimRight(ToUnixSlash(ExcludeDir), "/")

	if SourceDir == "" {
		FatalError("bootstrap", NewError("--source-dir= cannot be empty"))
	}

	if !Exists(SourceDir) {
		FatalError("bootstrap", NewError("--source-dir= does not exist"))
	}

	if TargetDir == "" {
		FatalError("bootstrap", NewError("--target-dir= cannot be empty"))
	} else {
		if !Exists(TargetDir) {
			MakeDirs(TargetDir)
		}
	}

	if ExcludeDir != "" {
		if !Exists(ExcludeDir) {
			FatalError("bootstrap", NewError("--exclude-dir= does not exist"))
		}
	}

	if FileExt != "" {
		fextMatch = regexp.MustCompile("(?i)" + FileExt)
	}

	if MinSizeMB != -1 {
		MinSize = MinSizeMB << 20
	}
	if MaxSizeMB != -1 {
		MaxSize = MaxSizeMB << 20
	}

	if MinAge != "" {
		MinAgeUnix = TimeStr2Unix(MinAge)
	}
	if MaxAge != "" {
		MaxAgeUnix = TimeStr2Unix(MaxAge)
	}

	return nil
}
