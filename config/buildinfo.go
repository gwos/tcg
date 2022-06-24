package config

// Variables to control the build info
// can be overridden by Go linker during the build step:
// go build -ldflags "-X 'github.com/gwos/tcg/config.buildTag=<TAG>' -X 'github.com/gwos/tcg/config.buildTime=`date --rfc-3339=s`'"
var (
	buildTag  = "8.x.x"
	buildTime = "Build time not provided"
)

// BuildInfo describes the build properties
type BuildInfo struct {
	Tag  string `json:"tag"`
	Time string `json:"time"`
}

// GetBuildInfo returns the build properties
func GetBuildInfo() BuildInfo {
	return BuildInfo{buildTag, buildTime}
}
