/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"sync"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gofastcopy",
	Short: "",
	Long:  ``,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		Bootstrap()
		timeStart = GetNowUnixMilli()
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if IsMirror == true {
			taskMirrorCleanTargetDir()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			defer wg.Done()
			taskFastCopy()
		}()

		go func() {
			defer wg.Done()
			taskSelectFiles()
		}()

		wg.Wait()

	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		timeStop = GetNowUnixMilli()
		tDuration := float64(float64(timeStop-timeStart) / float64(1000.0))
		showSpeed(tDuration)
		PrintlnInfo("purple", "Time Duraton", tDuration, " sec")
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
	rootCmd.PersistentFlags().StringVar(&SourceDir, "source-dir", "", "source folder")
	rootCmd.PersistentFlags().StringVar(&TargetDir, "target-dir", "", "destination folder")
	rootCmd.PersistentFlags().StringVar(&ExcludeDir, "exclude-dir", "", "will not copy the file if it is in the exclude-dir")
	//
	rootCmd.PersistentFlags().BoolVar(&IsSerial, "serial", false, "optimization for hard disk, not for ssd")
	rootCmd.PersistentFlags().IntVar(&CopyMode, "copy-mode", 0, "mode: 0 = zero copy, 1 = simd copy,2 = slow copy, default = 0")
	//
	rootCmd.PersistentFlags().BoolVar(&IsIgnoreDotFile, "ignore-dot-file", false, "ignore the file if its file name starts with dot(.), i.e.: .DS_Store")
	rootCmd.PersistentFlags().BoolVar(&IsIgnoreEmptyFolder, "ignore-empty-folder", false, "ignore the folder if it contains nothing")
	rootCmd.PersistentFlags().BoolVar(&IsFollowSymlink, "follow-symlink", false, "true: copy linked-file, false: copy soft-link only.")
	rootCmd.PersistentFlags().BoolVar(&IsOverwrite, "overwrite", true, "allow to overwrite the existing files")
	rootCmd.PersistentFlags().BoolVar(&IsMirror, "mirror", false, "delete files in --target-dir but NOT in --source-dir")
	//
	rootCmd.PersistentFlags().StringVar(&FileExt, "ext", "", "file type filter, i.e.: .mp4 or .png or .jpg ... ")
	//
	rootCmd.PersistentFlags().Int64Var(&MinSize, "min-size", -1, "from the minimum file size")
	rootCmd.PersistentFlags().Int64Var(&MaxSize, "max-size", -1, "to the maximum file size")
	rootCmd.PersistentFlags().Int64Var(&MinSizeMB, "min-size-mb", -1, "i.e.: 16 means 16MB, will replace --min-size=16*1024*1024 automatically")
	rootCmd.PersistentFlags().Int64Var(&MaxSizeMB, "max-size-mb", -1, "i.e.: 32 means 32MB, will replace --max-size=32*1024*1024 automatically")
	//
	rootCmd.PersistentFlags().StringVar(&MinAge, "min-age", "", "format: \"2023-12-03 15:09:08\" or 2023-12-03,15:09:08 ")
	rootCmd.PersistentFlags().StringVar(&MaxAge, "max-age", "", "format: \"2023-12-25 23:59:59\" or 2023-12-25,23:59:59 ")

}
