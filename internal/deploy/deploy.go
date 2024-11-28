package deploy

import (
	"fmt"
	"strings"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/docker"
	"github.com/bjarneo/pipe/internal/logger"
	"github.com/bjarneo/pipe/internal/ssh"
)

// Deploy performs the main deployment process
func Deploy(cfg *config.Config, log *logger.Logger) error {
	// Log start of deployment
	if err := log.Info("Starting deployment process"); err != nil {
		return err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Preliminary checks
	if err := docker.Check(cfg, log); err != nil {
		return err
	}

	if err := ssh.Check(cfg, log); err != nil {
		return err
	}

	// Build Docker image
	if err := docker.Build(cfg, log); err != nil {
		return err
	}

	// Transfer Docker image
	if err := docker.Transfer(cfg, log); err != nil {
		return err
	}

	// Copy environment file if it exists
	if cfg.EnvFile != "" {
		if err := copyEnvFile(cfg, log); err != nil {
			return err
		}
	}

	// Deploy container
	if err := docker.Deploy(cfg, log); err != nil {
		return err
	}

	return log.Info("Deployment completed successfully! 🚀")
}

// Rollback performs a rollback to the previous version
func Rollback(cfg *config.Config, log *logger.Logger) error {
	if err := log.Info("Starting rollback process..."); err != nil {
		return err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Check SSH connection
	if err := ssh.Check(cfg, log); err != nil {
		return err
	}

	// Get current container image
	getCurrentImageCmd := fmt.Sprintf("%s \"docker inspect --format='{{.Config.Image}}' %s\"",
		ssh.GetCommand(cfg), cfg.ContainerName)
	result, err := ssh.ExecuteCommand(log, getCurrentImageCmd, "Getting current container information")
	if err != nil {
		return fmt.Errorf("failed to get current container information: %v", err)
	}
	currentImage := strings.TrimSpace(result.Stdout)

	// Get image history sorted by creation time
	getImagesCmd := fmt.Sprintf("%s \"docker images %s --format '{{.Repository}}:{{.Tag}}___{{.CreatedAt}}' | sort -k2 -r\"",
		ssh.GetCommand(cfg), cfg.Image)
	history, err := ssh.ExecuteCommand(log, getImagesCmd, "Getting image history")
	if err != nil {
		return fmt.Errorf("failed to get image history: %v", err)
	}

	// Parse and sort images by creation time
	images := strings.Split(strings.TrimSpace(history.Stdout), "\n")
	if len(images) < 2 {
		return fmt.Errorf("no previous version found to rollback to")
	}

	// Find current image and get the next one
	var previousImage string
	for i, img := range images {
		imageName := strings.Split(img, "___")[0]
		if imageName == currentImage && i+1 < len(images) {
			previousImage = strings.Split(images[i+1], "___")[0]
			break
		}
	}

	if previousImage == "" {
		return fmt.Errorf("could not find previous version to rollback to")
	}

	// Log the versions involved
	if err := log.Info(fmt.Sprintf("Found previous version: %s", previousImage)); err != nil {
		return err
	}

	if err := performRollback(cfg, log, previousImage); err != nil {
		return err
	}

	// Clean up backup container
	cleanupCmd := fmt.Sprintf("%s \"docker rm %s_backup\"", ssh.GetCommand(cfg), cfg.ContainerName)
	_, _ = ssh.ExecuteCommand(log, cleanupCmd, "Cleaning up backup container")

	return log.Info("Rollback completed successfully! 🔄")
}

// copyEnvFile copies the environment file to the remote host
func copyEnvFile(cfg *config.Config, log *logger.Logger) error {
	copyEnvCmd := fmt.Sprintf("scp %s %s %s@%s:~/%s",
		ssh.GetKeyFlag(cfg), cfg.EnvFile, cfg.User, cfg.Host, cfg.EnvFile)
	_, err := ssh.ExecuteCommand(log, copyEnvCmd, "Copying environment file to server")
	return err
}

// performRollback executes the rollback operation
func performRollback(cfg *config.Config, log *logger.Logger, previousImage string) error {
	envFileFlag := ""
	if cfg.EnvFile != "" {
		envFileFlag = fmt.Sprintf("--env-file ~/%s", cfg.EnvFile)
	}

	rollbackCommands := strings.Join([]string{
		// Stop and rename current container (for backup)
		fmt.Sprintf("docker stop %s", cfg.ContainerName),
		fmt.Sprintf("docker rename %s %s_backup", cfg.ContainerName, cfg.ContainerName),

		// Start container with previous version
		fmt.Sprintf("docker run -d --name %s --restart unless-stopped -p %s:%s %s %s",
			cfg.ContainerName, cfg.HostPort, cfg.ContainerPort,
			envFileFlag, previousImage),
	}, " && ")

	// Execute rollback
	rollbackCmd := fmt.Sprintf("%s \"%s\"", ssh.GetCommand(cfg), rollbackCommands)
	if _, err := ssh.ExecuteCommand(log, rollbackCmd, "Rolling back to previous version"); err != nil {
		// If rollback fails, attempt to restore the backup
		if restoreErr := restoreBackup(cfg, log); restoreErr != nil {
			return fmt.Errorf("rollback failed and restore failed: %v (original error: %v)", restoreErr, err)
		}
		return fmt.Errorf("rollback failed, restored previous version: %v", err)
	}

	// Verify new container is running
	verifyCmd := fmt.Sprintf("%s \"docker ps --filter name=%s --format '{{.Status}}'\"",
		ssh.GetCommand(cfg), cfg.ContainerName)
	result, err := ssh.ExecuteCommand(log, verifyCmd, "Verifying rollback container status")
	if err != nil {
		return err
	}

	if !strings.Contains(result.Stdout, "Up") {
		// If verification fails, attempt to restore the backup
		if restoreErr := restoreBackup(cfg, log); restoreErr != nil {
			return fmt.Errorf("rollback verification failed and restore failed: %v", restoreErr)
		}
		return fmt.Errorf("rollback verification failed, restored previous version")
	}

	return nil
}

// restoreBackup attempts to restore the backup container
func restoreBackup(cfg *config.Config, log *logger.Logger) error {
	restoreCmd := fmt.Sprintf("%s \"docker stop %s || true && docker rm %s || true && docker rename %s_backup %s && docker start %s\"",
		ssh.GetCommand(cfg), cfg.ContainerName, cfg.ContainerName,
		cfg.ContainerName, cfg.ContainerName, cfg.ContainerName)
	_, err := ssh.ExecuteCommand(log, restoreCmd, "Restoring previous version after failed rollback")
	return err
} 