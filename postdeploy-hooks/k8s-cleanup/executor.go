package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// CommandExecutor contains command execution information.
type CommandExecutor struct {
	// BinPath is the path of the binary being used for the command (e.g. the path
	// to the kubectl binary if the kubectl command is to be used).
	binPath string
}

// CreateCommandExecutor returns a CommandExecutor for the given binary.
func CreateCommandExecutor(binPath string) *CommandExecutor {
	ce := &CommandExecutor{
		binPath: binPath,
	}
	return ce
}

// execCommand runs the given command and returns the output.
func (ce CommandExecutor) execCommand(args []string) (string, error) {
	fmt.Printf("Running the following command: %s %s\n", ce.binPath, args)
	cmd := exec.Command(ce.binPath, args...)
	// By default set locations to standard error and output (visible in cloud build logs)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Write error output to two locations simultaneously. This allows seeing the error output
	// as the execution is happening (by writing to standard error) and also allows gathering
	// all stderr at the end (by also writing to var stderr).
	var stderr bytes.Buffer
	errWriter := io.MultiWriter(&stderr, cmd.Stderr)
	cmd.Stderr = errWriter

	// Write error output to two locations simultaneously. This allows seeing the output
	// as the execution is happening (by writing to stdout) and also allows gathering
	// all stdout at the end (by also writing to var stdout).
	var stdout bytes.Buffer
	outWriter := io.MultiWriter(&stdout, cmd.Stdout)
	cmd.Stdout = outWriter

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command, err is %w", err)
	}

	// Wait for everything to finish.
	if err := cmd.Wait(); err != nil {
		// Read the stdErr output
		errorOutput := stderr.Bytes()
		fullErr := fmt.Errorf("error running command: %w\n%s", err, errorOutput)
		return "", fullErr
	}
	return stdout.String(), nil
}
