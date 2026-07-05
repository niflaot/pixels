// Package build exposes process build metadata shared by binaries and tests.
package build

// Info describes the current emulator build.
type Info struct {
	Name    string
	Version string
}

// DefaultInfo returns the default build metadata for local development.
func DefaultInfo() Info {
	return Info{
		Name:    "pixels",
		Version: "dev",
	}
}
