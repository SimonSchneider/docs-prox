package shell

import (
	"context"
	"io"
	"os"
	"os/exec"
	"time"
)

// Config holds the config for the shell calls
type Config struct {
	timeout time.Duration
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
}

func newDefault() Config {
	return Config{
		timeout: 0,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Stdin:   os.Stdin,
	}
}

func (c Config) clone() Config {
	return Config{
		timeout: c.timeout,
		Stdout:  c.Stdout,
		Stderr:  c.Stderr,
		Stdin:   c.Stdin,
	}
}

// Out sets the Stdout for the shell call
func Out(writer io.Writer) Config {
	return newDefault().Out(writer)
}

// Out sets the Stdout for the shell call
func (c Config) Out(writer io.Writer) Config {
	n := c.clone()
	n.Stdout = writer
	return n
}

// Err sets the Stderr for the shell call
func Err(writer io.Writer) Config {
	return newDefault().AllOut(writer)
}

// Err sets the Stderr for the shell call
func (c Config) Err(writer io.Writer) Config {
	n := c.clone()
	n.Stderr = writer
	return n
}

// AllOut sets both Stdout and Stderr for the shell call
func AllOut(writer io.Writer) Config {
	return newDefault().AllOut(writer)
}

// AllOut sets both Stdout and Stderr for the shell call
func (c Config) AllOut(writer io.Writer) Config {
	return c.Out(writer).Err(writer)
}

// Timeout sets the timeout for a shell call
func Timeout(duration time.Duration) Config {
	return newDefault().Timeout(duration)
}

// Timeout sets the timeout for a shell call
func (c Config) Timeout(duration time.Duration) Config {
	n := c.clone()
	n.timeout = duration
	return n
}

// Run the given shell command with default config
func Run(command string, args ...string) error {
	return newDefault().Run(command, args...)
}

// Run the given shell command with the given config
func (c Config) Run(command string, args ...string) error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(c.timeout))
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	cmd.Stdin = c.Stdin
	return cmd.Run()
}
