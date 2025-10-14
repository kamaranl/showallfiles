# ShowAllFiles

ShowAllFiles is a Windows utility that allows users to quickly toggle the visibility of hidden files in File Explorer. It provides a system tray interface, supports global hotkeys, logging, and optional verbose console output. ShowAllFiles was inspired by the MacOS feature available in the Finder application.

[![latest release](https://badgen.net/github/release/kamaranl/showallfiles?icon=github&cache=3600)](https://github.com/kamaranl/showallfiles/releases/latest)

## Features

* Toggle hidden files visibility via system tray menu or global hotkey (`Win + Shift + .`).
* System tray integration with toggle, about, and quit menu items.
* Configurable logging (level and file output), with log rotation.
* Optional console for verbose output and debugging.
* Windows environment-aware, with automatic handling of system registry keys.
* Message box notifications for errors and information.

## Installation

Install the [latest version](https://github.com/kamaranl/showallfiles/releases) from the [releases](https://github.com/kamaranl/showallfiles/releases) page.

## Usage

Launch ShowAllFiles by double-clicking **ShowAllFiles.exe** and toggle the visibility of your hidden files via the tray icon or hotkey.

![demo](/docs/demo.gif)

ShowAllFiles can be run with optional command-line flags:

```text
Usage of ShowAllFiles.exe:
      --log-level string   Log level: DEBUG|INFO|WARN|ERROR|FATAL|PANIC (default "INFO")
      --log string         File path to save log output
  -v, --verbose            Allocates a new console for verbose output
      --version            Prints version to console
```

## Hotkeys

* `Win + Shift + .` : Toggles visibility of hidden files.

## System Tray

The application provides a system tray icon with the following options:

* **Show/Hide** : Show or hide hidden files.
* **About** : Display application version.
* **Quit** : Exit the application.

## Logging

ShowAllFiles uses `logrus` for logging and supports:

* File output with log rotation (4 backups, 28-day retention).
* Configurable log levels.
* Verbose output via console.

## Registry

ShowAllFiles interacts with the following Windows registry key:

```text
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced
```

Specifically, it toggles the `Hidden` value to show or hide hidden files.

## Development

* Written in Go for Windows (`//go:build windows`).
* Uses `systray` for system tray integration.
* Uses `hotkey` library for global hotkey support.
* Logging handled by `logrus` with optional `lumberjack` rotation.

## Notes

* Designed for Windows **only**.
* Requires environment variable `SystemRoot` to be set.
* Base image of the folder icons used in this project were created by [kmg design](https://www.flaticon.com/authors/kmg-design) on [Flaticon](https://www.flaticon.com)
