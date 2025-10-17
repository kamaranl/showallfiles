<div align="center">

<h1>ShowAllFiles</h1>

<p>Show hidden files in the File Explorer</p>

<img src="docs/banner.png" alt="">

[![latest-release](https://badgen.net/github/release/kamaranl/showallfiles?icon=github&cache-3600)](https://github.com/kamaranl/showallfiles/releases/latest)
[![license](https://badgen.net/static/license/BSD-3-Clause/blue?cache-3600)](https://spdx.org/licenses/BSD-3-Clause.html)

</div>

## Overview

ShowAllFiles is a tray application for Windows that allows users to quickly toggle the visibility of hidden files in the File Explorer. It provides a system tray interface, supports global hotkeys, logging, and optional verbose console output. ShowAllFiles was inspired by the [MacOS feature](https://macos-defaults.com/finder/appleshowallfiles.html) available in the Finder application.

## Features

* Toggle hidden files visibility via system tray menu or global hotkey (`Win + Shift + .`).
* System tray integration.
* Configurable logging (level and file output), with log rotation.
* Optional console for verbose output and debugging.
* Windows environment-aware, with automatic handling of system registry keys.
* Message box notifications for errors and information.

## Installation

Install the [latest version](https://github.com/kamaranl/showallfiles/releases/latest) from the [releases](https://github.com/kamaranl/showallfiles/releases) page.

![install-demo](/docs/install.gif)

### Installer

Installation via the installer \(\***.zip**\) is the recommended method because it simplifies the process of installing ShowAllFiles to your device by:

* Installing Certificate Authorities to the `CurrentUser` certificate store.
* Installing ShowAllFiles to its own folder in `%LOCALAPPDATA%\Programs`.
* Creating an uninstaller.
* Creating a start menu shortcut.
* Enabling ShowAllFiles to run on device startup.
* Writing product information to the registry.

### Standalone

Tech-savvy users are welcome to download the standalone executable \(\***.exe**\), rename it, and add it to their `$env:PATH`.

## Usage

Toggle the visibility of your hidden files via the tray icon or the defined hotkey.

![usage-demo](/docs/usage.gif)

ShowAllFiles can also be ran with optional command-line flags:

```text
Usage of ShowAllFiles.exe:
      --log-level string   Log level: DEBUG|INFO|WARN|ERROR|FATAL|PANIC (default "INFO")
      --log string         File path to save log output
  -v, --verbose            Allocates a new console for verbose output
      --version            Prints version to console
```

## Components

### Hotkey

* `Win + Shift + .` : Toggles visibility of hidden files.

### System Tray

The application provides a system tray icon with the following options:

* **Show/Hide** : Show or hide hidden files.
* **About** : Display application version.
* **Report bug** : Opens the [issues](https://github.com/kamaranl/showallfiles/issues) page in the browser.
* **Quit** : Exit the application.

### Logging

ShowAllFiles uses `logrus` for logging and supports:

* File output with log rotation (4 backups, 28-day retention).
* Configurable log levels.
* Verbose output via console.

### Registry

ShowAllFiles interacts with the following Windows registry key:

```text
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced
```

Specifically, it toggles the `Hidden` property value to show or hide hidden files.

## Remarks

* Designed and compiled for **Windows only**.
* Requires environment variable `SystemRoot` to be set.

## Acknowledgements

* Base image of the folder icon used in this project was created by [kmg design](https://www.flaticon.com/authors/kmg-design) on [Flaticon](https://www.flaticon.com)
