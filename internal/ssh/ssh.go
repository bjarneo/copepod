package ssh

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bjarneo/copepod/internal/config"
	"github.com/bjarneo/copepod/internal/logger"
)

// CommandResult contains the output of a command
type CommandResult struct {
	Stdout string
	Stderr string
}

// GetKeyFlag returns the SSH key flag if SSHKey is set
func GetKeyFlag(cfg *config.Config) string {
	if cfg.SSHKey != "" {
		return fmt.Sprintf("-i %s", cfg.SSHKey)
	}
	return ""
}

// GetCommand returns the full SSH command with or without the key flag
func GetCommand(cfg *config.Config) string {
	sshKeyFlag := GetKeyFlag(cfg)
	if sshKeyFlag != "" {
		return fmt.Sprintf("ssh %s %s@%s", sshKeyFlag, cfg.User, cfg.Host)
	}
	return fmt.Sprintf("ssh %s@%s", cfg.User, cfg.Host)
}

// Check checks SSH connection to the remote host
func Check(cfg *config.Config, log *logger.Logger) error {
	command := fmt.Sprintf("%s echo \"SSH connection successful\"", GetCommand(cfg))
	_, err := ExecuteCommand(log, command, "Checking SSH connection")
	return err
}

// ExecuteCommand executes a shell command and streams the output
func ExecuteCommand(log *logger.Logger, command string, description string) (*CommandResult, error) {
	if err := log.Info(fmt.Sprintf("%s...", description)); err != nil {
		return nil, err
	}
	if err := log.Info(fmt.Sprintf("Executing: %s", command)); err != nil {
		return nil, err
	}

	cmd := exec.Command("sh", "-c", command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}

	var stdoutBuilder, stderrBuilder strings.Builder

	// Read stdout in real-time
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			stdoutBuilder.WriteString(line + "\n")
		}
	}()

	// Read stderr in real-time
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "error") || strings.Contains(line, "Error") {
				fmt.Println("ERROR:", line)
				stderrBuilder.WriteString(line + "\n")
			} else {
				fmt.Println(line)
				stdoutBuilder.WriteString(line + "\n")
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			return nil, fmt.Errorf("command failed with exit code %d: %v", exitErr.ExitCode(), err)
		}
		return nil, fmt.Errorf("command failed: %v", err)
	}

	result := &CommandResult{
		Stdout: stdoutBuilder.String(),
		Stderr: stderrBuilder.String(),
	}

	return result, nil
} 