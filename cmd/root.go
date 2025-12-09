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
	rootCmd.PersistentFlags().BoolVar(&IsWithLimitMemory, "with-limit-memory", false, "")
	rootCmd.PersistentFlags().BoolVar(&IsWithTimeUTC, "with-time-utc", false, "")
	rootCmd.PersistentFlags().BoolVar(&IsInSingleHDD, "in-single-hdd", false, "")
	//
	rootCmd.Flags().BoolVar(&IsIgnoreDotFile, "ignore-dot-file", true, "")
	rootCmd.Flags().BoolVar(&IsIgnoreEmptyFolder, "ignore-empty-folder", true, "")
	rootCmd.Flags().BoolVar(&IsOverwrite, "overwrite", false, "")
	//
	rootCmd.Flags().StringVar(&SourceDir, "source-dir", "", "")
	rootCmd.Flags().StringVar(&TargetDir, "target-dir", "", "")
	//
	rootCmd.Flags().StringVar(&ExcludeDir, "exclude-dir", "", "")
	rootCmd.Flags().StringVar(&FileExt, "file-ext", "", "")
	//
	rootCmd.Flags().Int64Var(&MinSize, "min-size", -1, "")
	rootCmd.Flags().Int64Var(&MaxSize, "max-size", -1, "")
	//
	rootCmd.Flags().StringVar(&MinAge, "min-age", "", "format: 20231203150908, means 2023-12-03 15:09:08")
	rootCmd.Flags().StringVar(&MaxAge, "max-age", "", "format: 20231225235959, means 2023-12-25 23:59:59")
	//
	rootCmd.Flags().IntVar(&ThreadNum, "threads", 0, "")
	//
}
