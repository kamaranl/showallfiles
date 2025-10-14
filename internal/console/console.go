// Copyright (c) 2025, Kamaran Layne <kamaran@layne.dev>
// See LICENSE for licensing information

// Package console provides functionality for attaching, detaching, spawning, and managing Windows console input and output streams.
// It allows binding the current process to an existing console, allocating a new console, and restoring original standard IO streams.
// The package handles preservation of original IO streams and ensures proper resource management when switching between console states.
// It is primarily intended for use in Windows environments where direct manipulation of console handles is required.
package console

import (
	"errors"
	"fmt"
	"os"

	"github.com/kamaranl/winapi"
)

var (
	// ErrBoundGuard is returned when an attempt is made to attach or spawn a console
	// while the Console instance is already bound to one.
	ErrBoundGuard = errors.New("console is already bound")

	// ErrNotBound is returned when an attempt is made to detach or operate on a console
	// that has not been attached or spawned yet.
	ErrNotBound = errors.New("console is not bound")
)

var (
	stdin, stdout, stderr *os.File
	preserved             bool
)

// Console represents a Windows console bound to the current process.
// It allows attaching to an existing console, spawning a new one, or freeing the console,
// and manages the associated input and output streams.
type Console struct {
	infile, outfile *os.File
	bound, debug    bool
}

// New creates a new Console instance and preserves the original standard IO streams.
// If debug is true, console operations will be skipped.
func New(debugger bool) *Console {
	preserveIO()
	return &Console{debug: debugger}
}

// Attach binds the Console to an existing Windows console. If a PID is provided,
// it attaches to that process's console; otherwise, it attaches to the parent process console.
// Returns ErrBoundGuard if the Console is already bound.
func (c *Console) Attach(pid ...uint32) error {
	if c.debug {
		return nil
	}
	if c.bound {
		return ErrBoundGuard
	}

	var procId winapi.ProcId
	if len(pid) > 0 {
		procId = winapi.ProcId(pid[0])
	} else {
		procId = winapi.ATTACH_PARENT_PROCESS
	}
	if err := winapi.AttachConsole(procId); err != nil {
		return err
	}
	if err := c.launchConsole(); err != nil {
		return err
	}

	fmt.Print("\r\033[K") // clear line
	return nil
}

// Detach restores the original standard IO streams, closes the console files,
// and frees the console if one is bound. Returns ErrNotBound if no console is attached.
func (c *Console) Detach() error {
	if c.debug {
		return nil
	}
	if !c.bound {
		return ErrNotBound
	}

	os.Stdin = stdin
	os.Stdout = stdout
	os.Stderr = stderr

	_ = c.infile.Close()
	_ = c.outfile.Close()

	c.infile, c.outfile = nil, nil
	c.bound = false

	return c.Free()
}

// Free detaches the Console from any Windows console without restoring IO streams.
// Returns an error if the underlying FreeConsole operation fails.
func (c *Console) Free() error {
	if c.debug {
		return nil
	}

	return winapi.FreeConsole()
}

// Spawn allocates a new Windows console and binds the Console instance to it.
// Returns ErrBoundGuard if the Console is already bound.
func (c *Console) Spawn() error {
	if c.debug {
		return nil
	}
	if c.bound {
		return ErrBoundGuard
	}
	if err := winapi.AllocConsole(); err != nil {
		return err
	}

	return c.launchConsole()
}

// bindConsole assigns a Windows standard handle (stdin, stdout, stderr) to the given file.
// Returns an error if the operation fails.
func (c *Console) bindConsole(name string, hstd winapi.HSTDIO, file *os.File) error {
	if err := winapi.SetStdHandle(hstd, file.Fd()); err != nil {
		return fmt.Errorf("failed to bind %s to %q: %v", name, file.Name(), err)
	}

	return nil
}

// launchConsole initializes and binds the console input and output streams for the Console instance.
// It opens the Windows console input ("CONIN$") and output ("CONOUT$") files, binds them to the
// standard handles (stdin, stdout, stderr), and updates the Console's internal file references.
// If any step fails, it returns a descriptive error and ensures resources are properly closed.
// On success, it sets the Console as bound and replaces the global os.Stdin, os.Stdout, and os.Stderr
// with the newly opened console files.
func (c *Console) launchConsole() error {
	in := "CONIN$"
	infile, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open %q: %v", in, err)
	}

	out := "CONOUT$"
	outfile, err := os.OpenFile(out, os.O_RDWR, 0)
	if err != nil {
		_ = infile.Close()

		return fmt.Errorf("failed to open %q: %v", out, err)
	}

	errs := []error{
		c.bindConsole("stdin", winapi.STD_INPUT_HANDLE, infile),
		c.bindConsole("stdout", winapi.STD_OUTPUT_HANDLE, outfile),
		c.bindConsole("stderr", winapi.STD_ERROR_HANDLE, outfile),
	}
	if err = errors.Join(errs...); err != nil {
		_ = infile.Close()
		_ = outfile.Close()

		return err
	}

	c.infile, c.outfile = infile, outfile
	os.Stdin, os.Stdout, os.Stderr = infile, outfile, outfile
	c.bound = true

	return nil
}

// preserveIO saves the current standard input, output, and error streams
// to global variables if they have not already been preserved. This ensures
// that the original IO streams can be restored or referenced later in the program.
func preserveIO() {
	if preserved {
		return
	}

	stdin, stdout, stderr = os.Stdin, os.Stdout, os.Stderr
	preserved = true
}
