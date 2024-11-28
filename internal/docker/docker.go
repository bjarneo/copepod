package docker

import (
	"fmt"
	"os"
	"strings"

	"github.com/bjarneo/copepod/internal/config"
	"github.com/bjarneo/copepod/internal/logger"
	"github.com/bjarneo/copepod/internal/ssh"
)

// Check checks if Docker is installed and running locally and remotely
func Check(cfg *config.Config, log *logger.Logger) error {
	// Check local Docker
	if _, err := ssh.ExecuteCommand(log, "docker info", "Checking local Docker installation"); err != nil {
		return fmt.Errorf("local Docker check failed: %v", err)
	}

	// Check remote Docker
	remoteCmd := fmt.Sprintf("%s \"docker info\"", ssh.GetCommand(cfg))
	if _, err := ssh.ExecuteCommand(log, remoteCmd, "Checking remote Docker installation"); err != nil {
		return fmt.Errorf("remote Docker check failed - please ensure Docker is installed on %s: %v", cfg.Host, err)
	}

	return nil
}

// Build builds the Docker image
func Build(cfg *config.Config, log *logger.Logger) error {
	// Check if Dockerfile exists
	if _, err := os.Stat(cfg.Dockerfile); os.IsNotExist(err) {
		return fmt.Errorf("%s not found", cfg.Dockerfile)
	}

	// Build Docker image with build arguments
	buildCmd := fmt.Sprintf("docker build --platform %s", cfg.Platform)

	// Add build arguments to the command
	for key, value := range cfg.BuildArgs {
		buildCmd += fmt.Sprintf(" --build-arg %s=%s", key, value)
	}

	buildCmd += fmt.Sprintf(" -t %s:%s .", cfg.Image, cfg.Tag)

	_, err := ssh.ExecuteCommand(log, buildCmd, "Building Docker image")
	return err
}

// Transfer transfers the Docker image to the remote host
func Transfer(cfg *config.Config, log *logger.Logger) error {
	deployCmd := fmt.Sprintf("docker save %s:%s | gzip | %s docker load",
		cfg.Image, cfg.Tag, ssh.GetCommand(cfg))
	_, err := ssh.ExecuteCommand(log, deployCmd, "Transferring Docker image to server")
	return err
}

// Deploy deploys the container on the remote host
func Deploy(cfg *config.Config, log *logger.Logger) error {
	containerConfig := []string{
		"-d",
		"--name", cfg.ContainerName,
		"--restart", "unless-stopped",
		"-p", fmt.Sprintf("%s:%s", cfg.HostPort, cfg.ContainerPort),
	}

	if cfg.Network != "" {
		containerConfig = append(containerConfig, "--network", cfg.Network)
	}

	if cfg.CPUs != "" {
		containerConfig = append(containerConfig, "--cpus", cfg.CPUs)
	}

	if cfg.Memory != "" {
		containerConfig = append(containerConfig, "--memory", cfg.Memory)
	}

	for _, volume := range cfg.Volumes {
		containerConfig = append(containerConfig, "-v", volume)
	}

	if cfg.EnvFile != "" {
		containerConfig = append(containerConfig, fmt.Sprintf("--env-file ~/%s", cfg.EnvFile))
	}

	containerConfig = append(containerConfig, fmt.Sprintf("%s:%s", cfg.Image, cfg.Tag))

	remoteCommands := strings.Join([]string{
		fmt.Sprintf("docker stop %s || true", cfg.ContainerName),
		fmt.Sprintf("docker rm %s || true", cfg.ContainerName),
		fmt.Sprintf("docker run %s", strings.Join(containerConfig, " ")),
	}, " && ")

	// Execute remote commands
	restartCmd := fmt.Sprintf("%s \"%s\"", ssh.GetCommand(cfg), remoteCommands)
	if _, err := ssh.ExecuteCommand(log, restartCmd, "Restarting container on server"); err != nil {
		return err
	}

	return verifyContainer(cfg, log)
}

// verifyContainer verifies that the container is running
func verifyContainer(cfg *config.Config, log *logger.Logger) error {
	verifyCmd := fmt.Sprintf("%s \"docker ps --filter name=%s --format '{{.Status}}'\"",
		ssh.GetCommand(cfg), cfg.ContainerName)
	result, err := ssh.ExecuteCommand(log, verifyCmd, "Verifying container status")
	if err != nil {
		return err
	}

	if !strings.Contains(result.Stdout, "Up") {
		return fmt.Errorf("container failed to start properly")
	}

	return nil
} 