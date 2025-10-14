// Copyright (c) 2025, Kamaran Layne <kamaran@layne.dev>
// See LICENSE for licensing information

// Package app provides the main application logic for the ShowAllFiles utility.
// It manages initialization, configuration, logging, environment variables, and the system tray UI.
// The Application struct encapsulates the application's error channel, metadata, and library functions.
// Key features include:
//   - Command-line flag parsing for logging, verbosity, and version display.
//   - Environment variable handling for debugging and runtime configuration.
//   - Logger setup with support for file output and log rotation.
//   - System tray integration with menu items for toggling hidden files, displaying about information, and quitting.
//   - Global hotkey registration for toggling hidden files visibility.
//   - Message box utilities for error and information dialogs.
//   - Console management for verbose output and debugging.
//
// The package is designed for Windows and interacts with the Windows registry and system APIs.
package app

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"github.com/kamaranl/showallfiles/internal/console"
	"github.com/kamaranl/showallfiles/internal/state"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.design/x/hotkey"
	"golang.org/x/sys/windows"
	"gopkg.in/natefinch/lumberjack.v2"
)

const regKeyPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`

const (
	statusVisible uint64 = iota + 1
	statusHidden
)

var (
	con  *console.Console
	log  *logrus.Logger
	flag struct {
		LogFile  string
		LogLevel string
		Verbose  bool
		Version  bool
	}
	env   map[string]string
	debug bool

	//go:embed icons/ShowAllFiles1.ico
	icoVisible []byte

	//go:embed icons/ShowAllFiles2.ico
	icoHidden []byte
)

// LogFormatter is a custom log formatter that embeds logrus.TextFormatter,
// allowing for additional customization of log output formatting.
type LogFormatter struct{ logrus.TextFormatter }

// Format formats a logrus.Entry by replacing all double quotes in the message with single quotes,
// then delegates formatting to the embedded TextFormatter. Returns the formatted log entry as a byte slice.
// If formatting fails, an error is returned.
func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Message = strings.ReplaceAll(entry.Message, `"`, `'`)
	b, err := f.TextFormatter.Format(entry)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Application represents the main application structure, containing channels for error handling,
// a Library instance for managing library operations, and metadata such as the application's name, version, and license.
type Application struct {
	ErrCh chan error
	Lib   Library
	Meta  struct {
		License string
		Name    string
		Version string
	}
}

// New creates a new Application instance with the specified name.
// It initializes the error channel and associates a Library with the application.
// Returns a pointer to the newly created Application.
func New(name string) *Application {
	app := &Application{
		ErrCh: make(chan error),
	}
	app.Meta.Name = name
	app.Lib = Library{App: app}

	return app
}

// Run starts the main execution flow of the Application.
// It attaches the console, parses command-line arguments, handles version display,
// checks for required environment variables, sets up logging, and launches the system tray.
// If invalid arguments or missing environment variables are detected, it displays appropriate
// error messages and exits the application.
func (a *Application) Run() {
	_ = con.Attach()

	if pflag.Arg(0) != "" {
		pflag.Usage()

		if !strings.EqualFold(pflag.Arg(0), "help") && pflag.Arg(0) != "?" {
			fmt.Fprintf(os.Stderr, "unknown arg: %s\n", pflag.Arg(0))
		}

		os.Exit(2)
	}
	if flag.Version {
		fmt.Fprintln(os.Stderr, a.Meta.Version)
		os.Exit(1)
	}
	if env["SystemRoot"] == "" {
		msg := `Environment variable "SystemRoot" not set`
		fmt.Fprintln(os.Stderr, msg)
		msgbox("Fatal Error", msg, windows.MB_OK|windows.MB_ICONERROR, 1)
	}

	setLogger(a.Meta.Name)
	log.Debug("Application ready")
	systray.Run(a.onReady, a.onExit)
}

// onReady initializes the application once it is ready to start.
// It sets up logging, registers a global hotkey for toggling hidden files,
// initializes systray menu items (toggle, about, quit), and starts watching
// for registry changes. The function enters a loop to handle menu item clicks
// and application errors, responding to user interactions and system events.
func (a *Application) onReady() {
	log.Info("Application started")

	hk := hotkey.New([]hotkey.Modifier{hotkey.ModWin, hotkey.ModShift}, hotkey.Key(windows.VK_OEM_PERIOD))
	if err := hk.Register(); err != nil {
		msg := fmt.Sprintf("Error registering global hotkey: %v", err)
		log.Fatal(msg)
		msgbox("Fatal Error", msg, windows.MB_OK|windows.MB_ICONERROR, 1)
	}

	go func() {
		for {
			<-hk.Keydown()
			log.Debug("Hotkey activated")
			a.Lib.ToggleHidden()
		}
	}()

	_, value, err := a.Lib.GetKeyValuePair(true)
	if err != nil {
		msg := fmt.Sprintf("Error fetching value of 'Hidden' during startup: %v", err)
		log.Fatal(msg)
		msgbox("Fatal Error", msg, windows.MB_OK|windows.MB_ICONERROR, 1)
	}
	state.Set("status_hidden", value)

	mToggle := systray.AddMenuItem("", "")
	state.Set("menu_toggle", mToggle)

	systray.AddSeparator()
	mTopAbout := systray.AddMenuItem("About", "")
	mTopReportBug := systray.AddMenuItem("Report bug", "")
	mTopQuit := systray.AddMenuItem("Quit", "")

	a.Lib.RefreshSystray()
	a.Lib.WatchRegistryKey()

	for {
		select {
		case <-mToggle.ClickedCh:
			log.Debug("*Clicked Toggle*")
			a.Lib.ToggleHidden()

		case <-mTopAbout.ClickedCh:
			log.Debug("*Clicked About*")
			msgbox("About",
				a.Meta.Name+", version "+a.Meta.Version+" ("+runtime.GOOS+"-"+runtime.GOARCH+")"+a.Meta.License,
				windows.MB_APPLMODAL|windows.MB_SETFOREGROUND, -1)

		case <-mTopReportBug.ClickedCh:
			log.Debug("*Clicked Report bug*")
			openUrl("https://github.com/kamaranl/showallfiles/issues")

		case <-mTopQuit.ClickedCh:
			log.Debug("*Clicked Quit*")
			systray.Quit()

		case err := <-a.ErrCh:
			log.Error(err)
		}
	}
}

// onExit handles cleanup operations when the application is stopping.
// It logs the application stop event, clears the application state,
// and if verbose mode is enabled, prints a countdown before exiting.
func (a *Application) onExit() {
	log.Info("Application stopped")
	state.Clear()

	if flag.Verbose {
		fmt.Println("This console will exit in")
		for i := 3; i > 0; i-- {
			fmt.Printf("%d...\n", i)
			time.Sleep(1 * time.Second)
		}
	}
}

// msgbox displays a Windows message box with the specified title, text, and box type.
// It ensures that only one message box with the same title is shown at a time by tracking state.
// The function runs the message box in a separate goroutine. If exitCode is non-negative,
// the application will exit with the provided exit code after the message box is closed.
//
// Parameters:
//
//	title    - The title of the message box window.
//	text:    - The message to display in the box.
//	boxtype  - The type of message box (e.g., MB_OK, MB_ICONERROR).
//	exitCode - If >= 0, exits the application with this code after closing the box.
func msgbox(title string, text string, boxtype uint32, exitCode int) {
	stateLabel := "msgbox_" + strings.ToLower(strings.ReplaceAll(title, " ", ""))
	if open, ok := state.Get[bool](stateLabel); ok && open {
		return
	}
	state.Set(stateLabel, true)

	go func() {
		_, _ = windows.MessageBox(
			0,
			windows.StringToUTF16Ptr(text),
			windows.StringToUTF16Ptr(title),
			windows.MB_APPLMODAL|boxtype,
		)
		state.Set(stateLabel, false)

		if exitCode >= 0 {
			os.Exit(exitCode)
		}
	}()
}

// openUrl launches the provided url in the default browser.
// It logs and displays errors when encountered; otherwise, no error means success.
func openUrl(url string) {
	log.Debugf("Launching %q", url)
	err := exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	if err != nil {
		msg := fmt.Sprintf("Error launching %q: %v", url, err)
		log.Error(msg)
		msgbox("Error", msg, windows.MB_OK|windows.MB_ICONERROR, -1)
	}
}

// setLogger initializes and configures the global logger instance.
// It sets the log formatter, log level, and output destinations based on the provided logName and global flag values.
// If a log file is specified, it validates the file path and configures log rotation using lumberjack.
// The logger output is set to both stderr and the log file (if valid).
// If verbose mode is enabled, it attempts to spawn a console window for logging output.
// Any errors encountered during setup are reported to stderr and, if applicable, via a message box.
func setLogger(logName string) {
	log = logrus.New()
	log.SetFormatter(&LogFormatter{logrus.TextFormatter{DisableColors: false, FullTimestamp: true}})

	if lvl, err := logrus.ParseLevel(flag.LogLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level: %v\n", err)
	} else {
		log.SetLevel(lvl)
	}

	writers := []io.Writer{}
	if flag.LogFile != "" {
		logF := flag.LogFile
		var logD, logN string

		info, err := os.Stat(logF)
		if err == nil && info.IsDir() {
			logD = logF
			logN = logName
		} else {
			logD = filepath.Dir(logF)
			logN = filepath.Base(logF)
		}

		logF = filepath.Join(logD, logN)
		logT := logF + ".TMP"
		valid := true

		f, err := os.Create(logT)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid log file: %v\n", err)
			valid = false
		} else {
			if err = f.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to close %q: %v\n", logT, err)
				valid = false
			}
			if err = os.Remove(logT); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to remove %q: %v\n", logT, err)
				valid = false
			}
		}

		if valid {
			writers = append(writers, &lumberjack.Logger{
				Filename:   logF,
				MaxBackups: 4,
				MaxAge:     28,
			})
			state.Set("log_file", logF)
		}
	}

	_ = con.Detach()

	if flag.Verbose {
		if err := con.Spawn(); err != nil {
			msg := fmt.Sprintf("Failed to spawn: %v", err)
			fmt.Fprintln(os.Stderr, msg)
			msgbox("Error", msg, windows.MB_OK|windows.MB_ICONERROR, 1)
		}
	}

	writers = append([]io.Writer{os.Stderr}, writers...)
	mw := io.MultiWriter(writers...)
	log.SetOutput(mw)
}

func init() {
	env = make(map[string]string)

	for _, key := range []string{"DEBUG", "SHOWALLFILES_CLI_ARGS", "SystemRoot", "TMP"} {
		if value, exists := os.LookupEnv(key); exists {
			env[key] = value
		}
	}

	debug = strings.EqualFold(env["DEBUG"], "true")
	con = console.New(debug)
	_ = con.Attach()

	if debug {
		if env["SHOWALLFILES_CLI_ARGS"] != "" {
			args := strings.Split(env["SHOWALLFILES_CLI_ARGS"], ";")
			os.Args = append([]string{os.Args[0]}, args...)
		}
	}

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", filepath.Base(os.Args[0]))
		pflag.PrintDefaults()
	}
	pflag.ErrHelp = errors.New("")
	pflag.CommandLine.SortFlags = false
	pflag.StringVar(&flag.LogLevel, "log-level", "INFO", "Log level: DEBUG|INFO|WARN|ERROR|FATAL|PANIC")
	pflag.StringVar(&flag.LogFile, "log", "", "File path to save log output")
	pflag.BoolVarP(&flag.Verbose, "verbose", "v", false, "Allocates a new console for verbose output")
	pflag.BoolVar(&flag.Version, "version", false, "Prints version")
	pflag.Parse()
}
