package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	IsDebug             bool
	IsIgnoreDotFile     bool
	IsIgnoreEmptyFolder bool
	IsOverwrite         bool
	IsPurge             bool
	MaxSize             int64
	MinSize             int64
	MaxSizeMB           int64
	MinSizeMB           int64
	MinAge              string
	MaxAge              string
	SourceDir           string
	TargetDir           string
	ExcludeDir          string
	FileExt             string
	//
	ThreadNum         int
	IsWithLimitMemory bool
	IsWithTimeUTC     bool
	IsWithMemStats    bool
	IsSerial          bool
	CopyMode          int
)

var (
	timeStart int64
	timeStop  int64
	numCPU    int = runtime.NumCPU()
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gofastcopy",
	Short: "./gofastcopy --source-dir=/path/to/folder-you-want-to-copy --target-dir=/path/to/target-folder ",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// positional args
		if SourceDir == "" {
			if len(args) == 1 || len(args) == 2 {
				SourceDir = args[0]
			}
		}

		if TargetDir == "" {
			if len(args) == 2 {
				TargetDir = args[1]
			}
		}
		// norm
		SourceDir = strings.TrimRight(ToUnixSlash(SourceDir), "/")
		TargetDir = strings.TrimRight(ToUnixSlash(TargetDir), "/")
		ExcludeDir = strings.TrimRight(ToUnixSlash(ExcludeDir), "/")

	},
	Run: func(cmd *cobra.Command, args []string) {
		argsValidate()
		fmt.Println(SEP)
		//
		timeStart = GetNowUnix()
		if IsPurge {
			purgeTargetDir()
		}
		//
		fastCopy()
		updateTargetDir()

	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		timeStop = GetNowUnix()
		if timeStart > 0 && timeStop > 0 {
			fmt.Printf("\n***** Elapse: %v (sec) *****\n", (timeStop - timeStart))
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&IsDebug, "debug", false, "if print debug info")
	//
	rootCmd.PersistentFlags().BoolVar(&IsWithLimitMemory, "with-mem-limit", false, "run with low momery, task will be forced to 4 threads")
	rootCmd.PersistentFlags().BoolVar(&IsWithTimeUTC, "with-time-utc", false, "use UTC timezone for parameter --min-age / --max-age")
	rootCmd.PersistentFlags().BoolVar(&IsWithMemStats, "with-mem-stats", false, "if print memory stats")
	rootCmd.PersistentFlags().BoolVar(&IsSerial, "serial", false, "optimization for hard disk, not for ssd")
	rootCmd.PersistentFlags().IntVar(&CopyMode, "copy-mode", 0, "mode: 0 = zero copy, 1 = simd copy, default = slow copy")
	//
	rootCmd.PersistentFlags().BoolVar(&IsIgnoreDotFile, "ignore-dot-file", false, "ignore the file if its file name starts with dot(.), i.e.: .DS_Store")
	rootCmd.PersistentFlags().BoolVar(&IsIgnoreEmptyFolder, "ignore-empty-folder", false, "ignore the folder if it contains nothing")
	rootCmd.PersistentFlags().BoolVar(&IsOverwrite, "overwrite", false, "allow to overwrite the existing files")
	rootCmd.PersistentFlags().BoolVar(&IsPurge, "purge", false, "delete files in --target-dir but NOT in --source-dir")

	//
	rootCmd.PersistentFlags().StringVar(&SourceDir, "source-dir", "", "source folder")
	rootCmd.PersistentFlags().StringVar(&TargetDir, "target-dir", "", "destination folder")
	//
	rootCmd.PersistentFlags().StringVar(&ExcludeDir, "exclude-dir", "", "will not copy the file if it is in the exclude-dir")
	rootCmd.PersistentFlags().StringVar(&FileExt, "ext", "", "file type filter, i.e.: .mp4 or .png or .jpg ... ")
	//
	rootCmd.PersistentFlags().Int64Var(&MinSize, "min-size", -1, "from the minimum file size")
	rootCmd.PersistentFlags().Int64Var(&MaxSize, "max-size", -1, "to the maximum file size")
	rootCmd.PersistentFlags().Int64Var(&MinSizeMB, "min-size-mb", -1, "i.e.: 16 means 16MB, will replace --min-size=16*1024*1024 automatically")
	rootCmd.PersistentFlags().Int64Var(&MaxSizeMB, "max-size-mb", -1, "i.e.: 32 means 32MB, will replace --max-size=32*1024*1024 automatically")
	//
	rootCmd.PersistentFlags().StringVar(&MinAge, "min-age", "", "format: 2023-12-03,15:09:08, means 2023-12-03 15:09:08")
	rootCmd.PersistentFlags().StringVar(&MaxAge, "max-age", "", "format: 2023-12-25,23:59:59, means 2023-12-25 23:59:59")
	//
	rootCmd.PersistentFlags().IntVar(&ThreadNum, "threads", 0, "force the concurrent tasks, more threads, more memory required")
	//
}
