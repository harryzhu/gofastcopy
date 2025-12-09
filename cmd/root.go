/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	IsDebug             bool
	IsIgnoreDotFile     bool
	IsIgnoreEmptyFolder bool
	IsOverwrite         bool
	MaxSize             int64
	MinSize             int64
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
	IsInSingleHDD     bool
	//
)

var (
	timeStart int64
	timeStop  int64
	numCPU    int = runtime.NumCPU()
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gofastcopy",
	Short: "",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		timeStart = GetNowUnix()

		SourceDir = ToUnixSlash(SourceDir)
		TargetDir = ToUnixSlash(TargetDir)
		ExcludeDir = ToUnixSlash(ExcludeDir)

		fmt.Println("-----")
		FlagsValidate()
		fmt.Println("-----")

	},
	Run: func(cmd *cobra.Command, args []string) {
		FastCopy()

		//
		timeStop = GetNowUnix()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if timeStart > 0 && timeStop > 0 {
			fmt.Printf("\n***** Elapse(sec): %v *****\n", (timeStop - timeStart))
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
	rootCmd.PersistentFlags().BoolVar(&IsWithLimitMemory, "with-limit-memory", false, "run with low momery, task will be forced to 4 threads")
	rootCmd.PersistentFlags().BoolVar(&IsWithTimeUTC, "with-time-utc", false, "use UTC timezone with parameter --min-age / --max-age")
	rootCmd.PersistentFlags().BoolVar(&IsInSingleHDD, "in-single-hdd", false, "only for massive small files copy on a single low speed hard disk")
	//
	rootCmd.Flags().BoolVar(&IsIgnoreDotFile, "ignore-dot-file", true, "ignore the file if its file name starts with dot(.), i.e.: .DS_Store")
	rootCmd.Flags().BoolVar(&IsIgnoreEmptyFolder, "ignore-empty-folder", true, "ignore the folder if it contains nothing")
	rootCmd.Flags().BoolVar(&IsOverwrite, "overwrite", false, "allow to overwrite the existing files")
	//
	rootCmd.Flags().StringVar(&SourceDir, "source-dir", "", "source folder")
	rootCmd.Flags().StringVar(&TargetDir, "target-dir", "", "destination folder")
	//
	rootCmd.Flags().StringVar(&ExcludeDir, "exclude-dir", "", "will not copy the file if it is in the exclude-dir")
	rootCmd.Flags().StringVar(&FileExt, "file-ext", "", "file type filter, i.e.: .mp4 or .png or .jpg ... ")
	//
	rootCmd.Flags().Int64Var(&MinSize, "min-size", -1, "from the minimum file size")
	rootCmd.Flags().Int64Var(&MaxSize, "max-size", -1, "to the maximum file size")
	//
	rootCmd.Flags().StringVar(&MinAge, "min-age", "", "format: 20231203150908, means 2023-12-03 15:09:08")
	rootCmd.Flags().StringVar(&MaxAge, "max-age", "", "format: 20231225235959, means 2023-12-25 23:59:59")
	//
	rootCmd.Flags().IntVar(&ThreadNum, "threads", 0, "force the concurrent tasks, more threads, more memory required")
	//
}
