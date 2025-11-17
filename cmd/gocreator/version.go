package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display detailed version information including commit hash and build date.`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("GoCreator v%s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", buildDate)
		fmt.Printf("Go version: %s\n", runtime.Version())
	},
}
