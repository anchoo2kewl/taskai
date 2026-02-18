package version

import (
	"runtime"
	"time"
)

var (
	// Version is the semantic version (set via ldflags during build)
	Version = "dev"

	// GitCommit is the git commit hash (set via ldflags during build)
	GitCommit = "unknown"

	// BuildTime is the build timestamp (set via ldflags during build)
	BuildTime = "unknown"
)

// Info holds version and build information
type Info struct {
	Version     string    `json:"version"`
	GitCommit   string    `json:"git_commit"`
	BuildTime   string    `json:"build_time"`
	GoVersion   string    `json:"go_version"`
	Platform    string    `json:"platform"`
	ServerTime  time.Time `json:"server_time"`
	DBVersion   int       `json:"db_version,omitempty"`
	Environment string    `json:"environment"`
}

// Get returns the current version information
func Get(env string, dbVersion int) Info {
	return Info{
		Version:     Version,
		GitCommit:   GitCommit,
		BuildTime:   BuildTime,
		GoVersion:   runtime.Version(),
		Platform:    runtime.GOOS + "/" + runtime.GOARCH,
		ServerTime:  time.Now().UTC(),
		DBVersion:   dbVersion,
		Environment: env,
	}
}
