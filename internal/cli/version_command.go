package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"runtime"

	"github.com/adams100111/agentic-learning-partner/internal/buildinfo"
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

func (a App) runVersion(args []string) int {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	asJSON := flags.Bool("json", false, "print version information as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	info := versionInfo{
		Version: buildinfo.Version,
		Commit:  buildinfo.Commit,
		Date:    buildinfo.Date,
		Go:      runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	if *asJSON {
		encoded, err := json.Marshal(info)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintln(a.Out, string(encoded))
		return 0
	}
	fmt.Fprintf(a.Out, "alp %s\ncommit: %s\nbuilt: %s\ngo: %s\nplatform: %s/%s\n",
		info.Version, info.Commit, info.Date, info.Go, info.OS, info.Arch)
	return 0
}
