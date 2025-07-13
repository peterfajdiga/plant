package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Process struct {
	Cmd    *exec.Cmd
	Stdin  io.Writer
	Stdout io.Reader
}

func Exec(command []string) (*Process, error) {
	cmd := exec.Command(command[0], command[1:]...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create StdinPipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create StdoutPipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to exec command %s: %w", command, err)
	}

	return &Process{
		Cmd:    cmd,
		Stdin:  stdin,
		Stdout: stdout,
	}, nil
}
