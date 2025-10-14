// Copyright (c) 2025, Kamaran Layne <kamaran@layne.dev>
// See LICENSE for licensing information

package app

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/getlantern/systray"
	"github.com/kamaranl/showallfiles/internal/state"
	"github.com/kamaranl/winapi"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// API defines the interface for interacting with Windows Explorer and system registry.
// It provides methods for retrieving registry key-value pairs, checking if a window is a file explorer,
// posting refresh messages, refreshing explorer windows and the system tray, toggling hidden files visibility,
// and watching for system messages and registry key changes. It also includes internal callback methods
// for enumerating windows and handling Windows event hooks.
type API interface {
	GetKeyValuePair(closeKey bool) (key registry.Key, value uint64, err error)
	IsFileExplorer(hwnd winapi.HWND) bool
	PostRefreshMessage(hwnd winapi.HWND)
	RefreshExplorerWindows()
	RefreshSystray()
	ToggleHidden()
	WatchMessageLoop()
	WatchRegistryKey()
	enumWindowsProc(hwnd winapi.HWND, lParam uintptr) uintptr
	winEventProc(evHook windows.Handle, ev uint32, hwnd winapi.HWND, objId, childId int32, evTId, evTime uint32)
}

// Library provides methods to interact with Windows File Explorer and system registry
// to toggle the visibility of hidden files, update the systray UI, and handle system events.
// It implements the API interface, which includes functions for registry access, window
// enumeration, message posting, and event watching.
//
// Methods:
//   - GetKeyValuePair: Retrieves the registry key and value for hidden files setting.
//   - IsFileExplorer: Determines if a window handle belongs to File Explorer.
//   - PostRefreshMessage: Posts a refresh command to a File Explorer window.
//   - RefreshExplorerWindows: Refreshes all open File Explorer windows.
//   - RefreshSystray: Updates the systray icon and menu based on hidden files status.
//   - ToggleHidden: Toggles the hidden files setting in the registry.
//   - WatchMessageLoop: Watches for foreground window changes to trigger refreshes.
//   - WatchRegistryKey: Watches for changes to the registry key controlling hidden files.
//   - enumWindowsProc: Callback for enumerating windows and posting refresh messages.
//   - winEventProc: Callback for handling system foreground events and refreshing Explorer.
//
// The Library type is designed for use in a Windows environment and relies on
// Windows API calls, registry access, and systray integration.
type Library struct {
	App *Application
	mu  sync.Mutex
}

// GetKeyValuePair opens a Windows registry key at the specified path and retrieves the value of the "Hidden" entry.
// If closeKey is true, the registry key will be closed before the function returns.
// It returns the opened registry key, the value of "Hidden" as a uint64, and an error if any operation fails.
func (l *Library) GetKeyValuePair(closeKey bool) (key registry.Key, value uint64, err error) {
	log.Debugf("Opening registry key %q", regKeyPath)
	key, err = registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return 0, 0, fmt.Errorf("failed call to OpenKey: %v", err)
	}
	if closeKey {
		defer func() { _ = key.Close() }()
	}

	log.Debug("Getting integer value of property 'Hidden'")
	value, _, err = key.GetIntegerValue("Hidden")
	if err != nil {
		return 0, 0, fmt.Errorf("failed call to GetIntegerValue: %v", err)
	}

	return key, value, nil
}

// IsFileExplorer determines whether the specified window handle (hwnd) belongs to a Windows File Explorer window.
// It checks the window class name for "CabinetWClass" and verifies that the associated process executable is "explorer.exe".
// Returns true if both conditions are met, indicating the window is a File Explorer; otherwise, returns false.
//
// Parameters:
//
//	hwnd - The window handle to test for a File Explorer window.
func (l *Library) IsFileExplorer(hwnd winapi.HWND) bool {
	classNameW := make([]uint16, syscall.MAX_PATH)
	if _, err := windows.GetClassName(hwnd, &classNameW[0], int32(len(classNameW))); err != nil {
		return false
	}

	className := windows.UTF16ToString(classNameW)
	if !strings.EqualFold(className, "CabinetWClass") {
		return false
	}
	log.Debug("Found window with class 'CabinetWClass'")

	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err != nil {
		return false
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		_ = windows.CloseHandle(handle)
		return false
	}

	exeNameW := make([]uint16, windows.MAX_PATH)
	size := uint32(len(exeNameW))
	if err = windows.QueryFullProcessImageName(handle, 0, &exeNameW[0], &size); err != nil {
		return false
	}
	_ = windows.CloseHandle(handle)

	exeName := filepath.Clean(windows.UTF16ToString(exeNameW))
	procName := filepath.Join(env["SystemRoot"], "explorer.exe")
	if strings.EqualFold(exeName, procName) {
		log.Debug("Found window for explorer.exe")
		return true
	}
	return false
}

// PostRefreshMessage posts a refresh command message to the specified window handle (hwnd).
// It sends a WM_COMMAND message with a predefined refresh identifier to trigger a refresh action
// in the target window. If posting the message fails, a warning is logged.
//
// Parameters:
//
//	hwnd - The window handle to which the refresh message will be posted.
func (l *Library) PostRefreshMessage(hwnd winapi.HWND) {
	log.Debugf("Posting refresh message to window handle %d", hwnd)
	if err := winapi.PostMessage(hwnd, winapi.WM_COMMAND, winapi.WPARAM(41504), 0); err != nil {
		log.Warnf("Could not post refresh message to window handle %d: %v", hwnd, err)
		return
	}
}

// RefreshExplorerWindows checks for open File Explorer windows and refreshes their state.
// If no File Explorer windows are found, it sets up a WinEventHook and starts a message loop
// to watch for new windows. The method is thread-safe and acquires a lock during execution.
// Logs warnings if window enumeration fails, and debug information about the current state.
func (l *Library) RefreshExplorerWindows() {
	l.mu.Lock()
	defer l.mu.Unlock()

	found := uint32(0)
	callback := windows.NewCallback(l.enumWindowsProc)
	defer runtime.KeepAlive(callback)

	log.Debug("Enumerating all available windows")
	if err := windows.EnumWindows(callback, unsafe.Pointer(&found)); err != nil {
		log.Warnf("Could not enumerate all available windows: %v", err)
		return
	}

	if found == 0 {
		log.Debug("File Explorer not currently open")
		if hook, ok := state.Get[windows.Handle]("hook_winEvent"); ok && hook != 0 {
			log.Debug("WinEvent hook is already set")
			return
		}

		l.WatchMessageLoop()
	}
}

// RefreshSystray updates the systray menu and icon based on the application's hidden status.
// It retrieves the toggle menu item and hidden status from the state, and adjusts the systray
// title, icon, and tooltip accordingly. If the required state values are not found, the function returns early.
func (l *Library) RefreshSystray() {
	log.Debug("Refreshing systray")
	toggle, ok := state.Get[*systray.MenuItem]("menu_toggle")
	if !ok {
		log.Error("Could not get state for 'menu_toggle': not set")
		return
	}

	hidden, ok := state.Get[uint64]("status_hidden")
	if !ok {
		log.Error("Could not get state for 'status_hidden': not set")
		return
	}
	if hidden == statusHidden {
		toggle.SetTitle("Show")
		systray.SetIcon(icoHidden)
		systray.SetTooltip(l.App.Meta.Name + " - Disabled")
	} else {
		toggle.SetTitle("Hide")
		systray.SetIcon(icoVisible)
		systray.SetTooltip(l.App.Meta.Name + " - Enabled")
	}
}

// ToggleHidden toggles the hidden status in the registry and updates the application state.
// It retrieves the current hidden status, switches it between visible and hidden,
// updates the registry key value accordingly, and sets the new state.
// If any error occurs during the process, it logs the error and returns.
func (l *Library) ToggleHidden() {
	key, value, err := l.GetKeyValuePair(false)
	if err != nil {
		log.Error(err)
		return
	}
	defer func() { _ = key.Close() }()

	var newValue uint64
	if value == statusHidden {
		newValue = statusVisible
	} else {
		newValue = statusHidden
	}

	log.Debug("Setting registry key value for property 'Hidden'")
	if err := key.SetDWordValue("Hidden", uint32(newValue)); err != nil {
		log.Errorf("Could not set registry key value: %v", err)
		return
	}
	state.Set("status_hidden", newValue)
}

// WatchMessageLoop starts a goroutine that sets a Windows event hook to monitor foreground window changes.
// It enters a message loop to process Windows messages, handling errors and cleanup appropriately.
// The hook and thread ID are stored in the application state for later reference.
// When the message loop exits (e.g., on WM_QUIT), the event hook is unregistered and state is cleaned up.
// Errors encountered during hook setup or message retrieval are sent to the provided error channel.
func (l *Library) WatchMessageLoop() {
	go func(errCh chan error) {
		log.Debug("Setting WinEvent hook")
		callback := windows.NewCallback(l.winEventProc)
		hook, err := winapi.SetWinEventHook(
			winapi.EVENT_SYSTEM_FOREGROUND,
			winapi.EVENT_SYSTEM_FOREGROUND,
			0,
			callback,
			0,
			0,
			winapi.WINEVENT_OUTOFCONTEXT,
		)
		if err != nil {
			errCh <- fmt.Errorf("failed call to SetWinEventHook: %v", err)
			return
		}

		state.Set("hook_winEvent", hook)
		state.Set("threadId_winEvent", windows.GetCurrentThreadId())

		log.Debug("Watching message loop")

		var msg winapi.MSG
		for {
			if r1, err := winapi.GetMessage(msg, 0, 0, 0); r1 == 0 {
				log.Debug("Received WM_QUIT")
				break
			} else if err != nil {
				errCh <- fmt.Errorf("failed call to GetMessage: %v", err)
				break
			}
			_ = winapi.TranslateMessage(msg)
			winapi.DispatchMessage(msg)
		}

		if hook != 0 {
			_ = winapi.UnhookWinEvent(hook)
		}

		state.Delete("hook_winEvent")
		state.Delete("threadId_winEvent")
	}(l.App.ErrCh)
}

// WatchRegistryKey starts a goroutine that monitors changes to a specific Windows registry key.
// It opens the registry key, sets up a notification event, and waits for changes to the key's value.
// When a change is detected, it retrieves the updated value, updates the application state,
// and refreshes the system tray and Explorer windows. Errors encountered during monitoring
// are sent to the application's error channel.
func (l *Library) WatchRegistryKey() {
	go func(errCh chan error) {
		log.Debugf("Retrieving handle for key %q", regKeyPath)
		var hKey windows.Handle
		if err := windows.RegOpenKeyEx(windows.HKEY_CURRENT_USER, windows.StringToUTF16Ptr(regKeyPath), 0, windows.KEY_NOTIFY, &hKey); err != nil {
			errCh <- fmt.Errorf("failed call to RegOpenKeyEx: %v", err)
			return
		}
		defer func() { _ = windows.RegCloseKey(hKey) }()

		log.Debugf("Creating RegNotify event")
		event, err := windows.CreateEvent(nil, 0, 0, nil)
		if err != nil {
			errCh <- fmt.Errorf("failed call to CreateEvent: %v", err)
			return
		}
		defer func() { _ = windows.CloseHandle(event) }()

		log.Debugf("Watching %q", regKeyPath)
		for {
			err = windows.RegNotifyChangeKeyValue(hKey, true, windows.REG_NOTIFY_CHANGE_LAST_SET, event, true)
			if err != nil {
				errCh <- fmt.Errorf("failed call to RegNotifyChangeKeyValue: %v", err)
				return
			}

			if r1, _ := windows.WaitForSingleObject(event, windows.INFINITE); r1 == windows.WAIT_OBJECT_0 {
				_, value, err := l.GetKeyValuePair(false)
				if err != nil {
					errCh <- fmt.Errorf("failed call to WaitForSingleObject: %v", err)
					return
				}
				state.Set("status_hidden", value)
				l.RefreshSystray()
				l.RefreshExplorerWindows()
			}
		}
	}(l.App.ErrCh)
}

// enumWindowsProc is a callback function used during window enumeration.
// It checks if the given window handle (hwnd) corresponds to a File Explorer window.
// If a File Explorer window is found and has not been previously marked as found,
// it sets the found flag to true and posts a refresh message to the window.
// The function always returns 1 to continue enumeration.
//
// Parameters:
//
//	hwnd   - The handle to the window being enumerated.
//	lParam - A pointer to a uint32 flag indicating whether a File Explorer window has been found.
//
// Returns:
//
//	uintptr - Always returns 1 to continue enumeration.
func (l *Library) enumWindowsProc(hwnd winapi.HWND, lParam uintptr) uintptr {
	foundPtr := (*uint32)(unsafe.Pointer(lParam))
	if l.IsFileExplorer(hwnd) {
		if *foundPtr == 0 {
			*foundPtr = 1
		}
		l.PostRefreshMessage(hwnd)
	}
	return 1
}

// winEventProc is a Windows event hook procedure for handling accessibility events.
// It checks if the event is associated with a File Explorer window and, if so,
// triggers a refresh message asynchronously after a short delay. If a thread ID
// is stored in the application state, it attempts to post a WM_QUIT message to
// that thread to signal termination. The function ignores events for non-root objects
// (objId != 0) and always returns 0 as required by the Windows event hook signature.
//
// Parameters:
//
//	eventHook     - Handle to the event hook.
//	event         - Event type identifier.
//	hwnd          - Handle to the window receiving the event.
//	objectId      - Object identifier for the event.
//	childId       - Child identifier for the event.
//	eventThreadId - Thread ID where the event occurred.
//	eventTime     - Timestamp of the event.
//
// Returns:
//
//	uintptr - Always returns 0.
func (l *Library) winEventProc(eventHook windows.Handle, event uint32, hwnd winapi.HWND, objectId, childId int32,
	eventThreadId, eventTime uint32,
) uintptr {
	if objectId != 0 {
		return 0
	}

	if l.IsFileExplorer(hwnd) {
		go func() {
			time.Sleep(500 * time.Millisecond)
			l.PostRefreshMessage(hwnd)

			if tID, ok := state.Get[uint32]("threadId_winEvent"); ok && tID != 0 {
				if err := winapi.PostThreadMessage(tID, winapi.WM_QUIT, 0, 0); err != nil {
					log.Warnf("Could not post WM_QUIT to thread %d: %v", tID, err)
				}
			}
		}()
	}
	return 0
}
