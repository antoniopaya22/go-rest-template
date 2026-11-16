// Package version exposes build metadata.
//
// Version, Commit and Date are set at link time:
//
//	go build -ldflags "-X .../internal/platform/version.Version=v1.2.3"
//
// When they are not set, the values embedded by the Go toolchain are used.
package version

import (
	"runtime"
	"runtime/debug"
)

// Build metadata injected with -ldflags.
var (
	Version = ""
	Commit  = ""
	Date    = ""
)

// Info describes the running build.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
}

// Get returns the build metadata of the running binary.
func Get() Info {
	info := Info{Version: Version, Commit: Commit, Date: Date, GoVersion: runtime.Version()}

	if build, ok := debug.ReadBuildInfo(); ok {
		if info.Version == "" && build.Main.Version != "" && build.Main.Version != "(devel)" {
			info.Version = build.Main.Version
		}
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = setting.Value
				}
			case "vcs.time":
				if info.Date == "" {
					info.Date = setting.Value
				}
			}
		}
	}

	if info.Version == "" {
		info.Version = "dev"
	}
	return info
}
