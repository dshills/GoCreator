package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// VersionInfo holds structured version and build information
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

var (
	// versionJSONFlag enables JSON output format for version information
	versionJSONFlag bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display detailed version information including version, commit hash, build date, and Go version.

Use --json flag to output version information in JSON format for scripting.`,
	Run: func(_ *cobra.Command, _ []string) {
		info := VersionInfo{
			Version:   version,
			Commit:    commit,
			BuildDate: buildDate,
			GoVersion: runtime.Version(),
			GOOS:      runtime.GOOS,
			GOARCH:    runtime.GOARCH,
		}

		if versionJSONFlag {
			outputJSON(&info)
		} else {
			outputText(&info)
		}
	},
}

// outputText prints version information in human-readable format
func outputText(info *VersionInfo) {
	fmt.Printf("GoCreator v%s\n", info.Version)
	fmt.Printf("Commit:      %s\n", info.Commit)
	fmt.Printf("Built:       %s\n", info.BuildDate)
	fmt.Printf("Go version:  %s\n", info.GoVersion)
	fmt.Printf("Platform:    %s/%s\n", info.GOOS, info.GOARCH)
}

// outputJSON prints version information in JSON format
func outputJSON(info *VersionInfo) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(info); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func setupVersionFlags() {
	versionCmd.Flags().BoolVar(&versionJSONFlag, "json", false, "Output version information as JSON")
}
