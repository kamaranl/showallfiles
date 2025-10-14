// Copyright (c) 2025, Kamaran Layne <kamaran@layne.dev>
// See LICENSE for licensing information

//go:build windows

//go:generate windres resource.rc -O coff -o resource.syso

// Package main provides the entry point for the ShowAllFiles application.
// It initializes the main application logic from the internal app package,
// embeds version information, and starts the application run loop.
package main

import (
	_ "embed"

	"github.com/kamaranl/showallfiles/internal/app"
)

const (
	// Name defines the application name used for display and logging purposes.
	Name = "ShowAllFiles"

	// License holds the license identifier and copyright notice for the application.
	License = `
Copyright © 2025, Kamaran Layne
BSD 3-Clause License

This software is distributed "as-is" with NO WARRANTY.
`
)

// Version holds the application version, embedded at build time from the VERSION file.
// It is used to display version information in the application and via command-line flags.
//
//go:embed VERSION
var Version string

// main is the entry point of the ShowAllFiles application.
// It creates a new Application instance, sets its version, and runs the application.
func main() {
	a := app.New(Name)
	a.Meta.Version = Version
	a.Meta.License = License
	a.Run()
}
