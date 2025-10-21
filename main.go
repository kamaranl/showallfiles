//go:build windows

// Copyright (c) 2025, Kamaran Layne <kamaran@layne.dev>
// See LICENSE for licensing information

/*
ShowAllFiles is a tray application for Windows that allows users to quickly
toggle the visibility of hidden files in the File Explorer. It provides a system
tray interface, supports global hotkeys, logging, and optional verbose console
output. ShowAllFiles was inspired by the MacOS feature available in the Finder
application.
*/
package main

import (
	_ "embed"

	"github.com/kamaranl/showallfiles/internal/app"
)

var (
	// Name defines the application name used for display and logging purposes.
	Name string = "ShowAllFiles"

	// License defines the application license and copyright notice. It is used
	// to display license information in the application.
	License string = "Copyright © 2025, Kamaran Layne\nBSD 3-Clause License"

	// Version holds the application version, embedded at build time from the
	// VERSION file. It is used to display version information in the
	// application and via command-line flags.
	//go:embed VERSION
	Version string
)

func main() {
	a := app.New(Name)
	a.Meta.Version = Version
	a.Meta.License = License
	a.Run()
}
