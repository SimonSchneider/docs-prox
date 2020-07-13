package shell

import (
	"context"
	"io"
	"os"
	"os/exec"
	"time"
)

type ShellData struct {
	timeout time.Duration
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
}

func newDefault() ShellData {
	return ShellData{
		timeout: 0,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Stdin:   os.Stdin,
	}
}

func (s ShellData) clone() ShellData {
	return ShellData{
		timeout: s.timeout,
		Stdout:  s.Stdout,
		Stderr:  s.Stderr,
		Stdin:   s.Stdin,
	}
}

func Out(writer io.Writer) ShellData {
	return newDefault().Out(writer)
}

func (s ShellData) Out(writer io.Writer) ShellData {
	n := s.clone()
	n.Stdout = writer
	return n
}

func Err(writer io.Writer) ShellData {
	return newDefault().AllOut(writer)
}

func (s ShellData) Err(writer io.Writer) ShellData {
	n := s.clone()
	n.Stderr = writer
	return n
}

func AllOut(writer io.Writer) ShellData {
	return newDefault().AllOut(writer)
}

func (s ShellData) AllOut(writer io.Writer) ShellData {
	return s.Out(writer).Err(writer)
}

func Timeout(duration time.Duration) ShellData {
	return newDefault().Timeout(duration)
}

func (s ShellData) Timeout(duration time.Duration) ShellData {
	n := s.clone()
	n.timeout = duration
	return n
}

func Run(command string, args ...string) error {
	return newDefault().Run(command, args...)
}

func (s ShellData) Run(command string, args ...string) error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(s.timeout))
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdout = s.Stdout
	cmd.Stderr = s.Stderr
	cmd.Stdin = s.Stdin
	return cmd.Run()
}
