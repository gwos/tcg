package config

type BuildVersion struct {
	Tag  string `json:"tag"`
	Time string `json:"time"`
}

var Version = BuildVersion{
	Tag:  "Version tag not provided",
	Time: "Build time not provided",
}
