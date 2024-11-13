package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config holds the deployment configuration
type Config struct {
	Host          string            `json:"host"`
	User          string            `json:"user"`
	Image         string            `json:"image"`
	Tag           string            `json:"tag"`
	Platform      string            `json:"platform"`
	SSHKey        string            `json:"sshKey"`
	ContainerName string            `json:"containerName"`
	ContainerPort string            `json:"containerPort"`
	HostPort      string            `json:"hostPort"`
	EnvFile       string            `json:"envFile"`
	Rollback      bool              `json:"rollback"`
	BuildArgs     map[string]string `json:"buildArgs"`
}

// Logger handles logging to both console and file
type Logger struct {
	file *os.File
}

// OSRelease contains OS information
type OSRelease struct {
	ID         string `json:"id"`
	VersionID  string `json:"version_id"`
	PrettyName string `json:"pretty_name"`
}

// CommandResult contains the output of a command
type CommandResult struct {
	Stdout string
	Stderr string
}

const helpText = `
Docker Deployment Tool

Usage:
  copepod [options]

Options:
  --host            Remote host to deploy to
  --user            SSH user for remote host
  --image           Docker image name (default: copepod_app)
  --tag             Docker image tag (default: latest)
  --platform        Docker platform (default: linux/amd64)
  --ssh-key         Path to SSH key (default: "")
  --container-name  Name for the container (default: copepod_app)
  --container-port  Container port (default: 3000)
  --host-port       Host port (default: 3000)
  --env-file        Environment file (default: "")
  --build-arg       Build arguments (can be specified multiple times, format: KEY=VALUE)
  --rollback        Rollback to the previous version
  --help            Show this help message

Environment Variables:
  DEPLOY_HOST      Remote host to deploy to
  DEPLOY_USER      SSH user for remote host
  DEPLOY_IMAGE     Docker image name
  DEPLOY_TAG       Docker image tag
  DEPLOY_PLATFORM  Docker platform
  SSH_KEY_PATH     Path to SSH key
  CONTAINER_NAME   Name for the container
  CONTAINER_PORT   Container port
  HOST_PORT        Host port
  ENV_FILE         Environment file
  BUILD_ARGS       Build arguments (comma-separated KEY=VALUE pairs)

Examples:
  copepod --host example.com --user deploy
  copepod --host example.com --user deploy --build-arg VERSION=1.0.0 --build-arg ENV=prod
  copepod --env-file .env.production --build-arg GIT_HASH=$(git rev-parse HEAD)
`

// NewLogger creates a new logger instance
func NewLogger(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: file}, nil
}

// Info logs an informational message
func (l *Logger) Info(message string) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] INFO: %s\n", timestamp, message)
	fmt.Print(message + "\n")
	_, err := l.file.WriteString(logMessage)
	return err
}

// Error logs an error message
func (l *Logger) Error(message string, err error) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	logMessage := fmt.Sprintf("[%s] ERROR: %s\n%s\n", timestamp, message, errStr)
	fmt.Printf("ERROR: %s\n", message)
	if err != nil {
		fmt.Printf("Error details: %s\n", err)
	}
	_, writeErr := l.file.WriteString(logMessage)
	return writeErr
}

// Close closes the log file
func (l *Logger) Close() error {
	return l.file.Close()
}

// arrayFlags allows for multiple flag values
type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// LoadConfig loads configuration from command line flags and environment variables
func LoadConfig() Config {
	var config Config
	var showHelp bool
	var buildArgs arrayFlags

	// Initialize BuildArgs map
	config.BuildArgs = make(map[string]string)

	// Define command line flags
	flag.StringVar(&config.Host, "host", getEnv("DEPLOY_HOST", ""), "Remote host to deploy to")
	flag.StringVar(&config.User, "user", getEnv("DEPLOY_USER", ""), "SSH user for remote host")
	flag.StringVar(&config.Image, "image", getEnv("DEPLOY_IMAGE", "app"), "Docker image name")
	flag.StringVar(&config.Tag, "tag", getEnv("DEPLOY_TAG", "latest"), "Docker image tag")
	flag.StringVar(&config.Platform, "platform", getEnv("DEPLOY_PLATFORM", "linux/amd64"), "Docker platform")
	flag.StringVar(&config.SSHKey, "ssh-key", getEnv("SSH_KEY_PATH", ""), "Path to SSH key")
	flag.StringVar(&config.ContainerName, "container-name", getEnv("CONTAINER_NAME", "app"), "Name for the container")
	flag.StringVar(&config.ContainerPort, "container-port", getEnv("CONTAINER_PORT", "3000"), "Container port")
	flag.StringVar(&config.HostPort, "host-port", getEnv("HOST_PORT", "3000"), "Host port")
	flag.StringVar(&config.EnvFile, "env-file", getEnv("ENV_FILE", ""), "Environment file")
	flag.Var(&buildArgs, "build-arg", "Build argument in KEY=VALUE format (can be specified multiple times)")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&config.Rollback, "rollback", false, "Rollback to previous version")

	// Custom usage message
	flag.Usage = func() {
		fmt.Println(helpText)
	}

	// Parse command line flags
	flag.Parse()

	// Show help if requested
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Process build arguments from command line
	for _, arg := range buildArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			config.BuildArgs[parts[0]] = parts[1]
		}
	}

	// Process build arguments from environment variable
	if envBuildArgs := os.Getenv("BUILD_ARGS"); envBuildArgs != "" {
		for _, arg := range strings.Split(envBuildArgs, ",") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				config.BuildArgs[parts[0]] = parts[1]
			}
		}
	}

	// Expand home directory in SSH key path
	if strings.HasPrefix(config.SSHKey, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			config.SSHKey = filepath.Join(home, config.SSHKey[2:])
		}
	}

	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// ValidateConfig validates the configuration
func (c *Config) ValidateConfig() error {
	if c.Host == "" || c.User == "" {
		return fmt.Errorf("missing required configuration: host and user must be provided")
	}
	return nil
}

// ExecuteCommand executes a shell command and logs the output
func ExecuteCommand(logger *Logger, command string, description string) (*CommandResult, error) {
	if err := logger.Info(fmt.Sprintf("%s...", description)); err != nil {
		return nil, err
	}
	if err := logger.Info(fmt.Sprintf("Executing: %s", command)); err != nil {
		return nil, err
	}

	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed: %s", description), err)
		return nil, err
	}

	result := &CommandResult{
		Stdout: string(output),
		Stderr: "",
	}

	if len(output) > 0 {
		logger.Info(fmt.Sprintf("Output: %s", string(output)))
	}

	return result, nil
}

// CheckDocker checks if Docker is installed and running
func CheckDocker(logger *Logger) error {
	_, err := ExecuteCommand(logger, "docker info", "Checking Docker installation")
	return err
}

// getSSHKeyFlag returns the SSH key flag if SSHKey is set
func getSSHKeyFlag(config *Config) string {
	if config.SSHKey != "" {
		return fmt.Sprintf("-i %s", config.SSHKey)
	}
	return ""
}

// getSSHCommand returns the full SSH command with or without the key flag
func getSSHCommand(config *Config) string {
	sshKeyFlag := getSSHKeyFlag(config)
	if sshKeyFlag != "" {
		return fmt.Sprintf("ssh %s %s@%s", sshKeyFlag, config.User, config.Host)
	}
	return fmt.Sprintf("ssh %s@%s", config.User, config.Host)
}

// CheckSSH checks SSH connection to the remote host
func CheckSSH(config *Config, logger *Logger) error {
	command := fmt.Sprintf("%s echo \"SSH connection successful\"", getSSHCommand(config))
	_, err := ExecuteCommand(logger, command, "Checking SSH connection")
	return err
}

// Rollback performs a rollback to the previous version by inspecting the current container
// TODO: fix this mess
// Rollback performs a rollback to the previous version using docker images history
func Rollback(config *Config, logger *Logger) error {
	if err := logger.Info("Starting rollback process..."); err != nil {
		return err
	}

	// Validate configuration
	if err := config.ValidateConfig(); err != nil {
		return err
	}

	// Check SSH connection
	if err := CheckSSH(config, logger); err != nil {
		return err
	}

	// Get current container image
	getCurrentImageCmd := fmt.Sprintf("%s \"docker inspect --format='{{.Config.Image}}' %s\"",
		getSSHCommand(config), config.ContainerName)
	result, err := ExecuteCommand(logger, getCurrentImageCmd, "Getting current container information")
	if err != nil {
		return fmt.Errorf("failed to get current container information: %v", err)
	}
	currentImage := strings.TrimSpace(result.Stdout)

	// Get image history sorted by creation time
	// Format: repository, tag, image ID, created, size
	getImagesCmd := fmt.Sprintf("%s \"docker images %s --format '{{.Repository}}:{{.Tag}}___{{.CreatedAt}}' | sort -k2 -r\"",
		getSSHCommand(config), config.Image)
	history, err := ExecuteCommand(logger, getImagesCmd, "Getting image history")
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
	if err := logger.Info(fmt.Sprintf("Found previous version: %s", previousImage)); err != nil {
		return err
	}

	// Prepare rollback commands
	envFileFlag := ""
	if config.EnvFile != "" {
		envFileFlag = fmt.Sprintf("--env-file ~/%s", config.EnvFile)
	}

	rollbackCommands := strings.Join([]string{
		// Stop and rename current container (for backup)
		fmt.Sprintf("docker stop %s", config.ContainerName),
		fmt.Sprintf("docker rename %s %s_backup", config.ContainerName, config.ContainerName),

		// Start container with previous version
		fmt.Sprintf("docker run -d --name %s --restart unless-stopped -p %s:%s %s %s",
			config.ContainerName, config.HostPort, config.ContainerPort,
			envFileFlag, previousImage),
	}, " && ")

	// Execute rollback
	rollbackCmd := fmt.Sprintf("%s \"%s\"", getSSHCommand(config), rollbackCommands)
	if _, err := ExecuteCommand(logger, rollbackCmd, "Rolling back to previous version"); err != nil {
		// If rollback fails, attempt to restore the backup
		restoreCmd := fmt.Sprintf("%s \"docker stop %s || true && docker rm %s || true && docker rename %s_backup %s && docker start %s\"",
			getSSHCommand(config), config.ContainerName, config.ContainerName,
			config.ContainerName, config.ContainerName, config.ContainerName)
		if _, restoreErr := ExecuteCommand(logger, restoreCmd, "Restoring previous version after failed rollback"); restoreErr != nil {
			return fmt.Errorf("rollback failed and restore failed: %v (original error: %v)", restoreErr, err)
		}
		return fmt.Errorf("rollback failed, restored previous version: %v", err)
	}

	// Verify new container is running
	verifyCmd := fmt.Sprintf("%s \"docker ps --filter name=%s --format '{{.Status}}'\"",
		getSSHCommand(config), config.ContainerName)
	result, err = ExecuteCommand(logger, verifyCmd, "Verifying rollback container status")
	if err != nil {
		return err
	}

	if !strings.Contains(result.Stdout, "Up") {
		// If verification fails, attempt to restore the backup
		restoreCmd := fmt.Sprintf("%s \"docker stop %s || true && docker rm %s || true && docker rename %s_backup %s && docker start %s\"",
			getSSHCommand(config), config.ContainerName, config.ContainerName,
			config.ContainerName, config.ContainerName, config.ContainerName)
		if _, restoreErr := ExecuteCommand(logger, restoreCmd, "Restoring previous version after failed verification"); restoreErr != nil {
			return fmt.Errorf("rollback verification failed and restore failed: %v", restoreErr)
		}
		return fmt.Errorf("rollback verification failed, restored previous version")
	}

	// Log the rollback details
	logger.Info(fmt.Sprintf("Successfully rolled back from %s to %s", currentImage, previousImage))

	// If everything is successful, remove the backup container
	cleanupCmd := fmt.Sprintf("%s \"docker rm %s_backup\"", getSSHCommand(config), config.ContainerName)
	_, _ = ExecuteCommand(logger, cleanupCmd, "Cleaning up backup container")

	return logger.Info("Rollback completed successfully! 🔄")
}

// Deploy performs the main deployment process
func Deploy(config *Config, logger *Logger) error {
	// Log start of deployment
	if err := logger.Info("Starting deployment process"); err != nil {
		return err
	}

	configJSON, _ := json.MarshalIndent(config, "", "  ")
	if err := logger.Info(fmt.Sprintf("Deployment configuration: %s", string(configJSON))); err != nil {
		return err
	}

	// Validate configuration
	if err := config.ValidateConfig(); err != nil {
		return err
	}

	// Preliminary checks
	if err := CheckDocker(logger); err != nil {
		return err
	}

	if err := CheckSSH(config, logger); err != nil {
		return err
	}

	// Check if Dockerfile exists
	if _, err := os.Stat("Dockerfile"); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found in current directory")
	}

	// Build Docker image with build arguments
	buildCmd := fmt.Sprintf("docker build --platform %s", config.Platform)

	// Add build arguments to the command
	for key, value := range config.BuildArgs {
		buildCmd += fmt.Sprintf(" --build-arg %s=%s", key, value)
	}

	buildCmd += fmt.Sprintf(" -t %s:%s .", config.Image, config.Tag)

	if _, err := ExecuteCommand(logger, buildCmd, "Building Docker image"); err != nil {
		return err
	}

	// Save and transfer Docker image
	deployCmd := fmt.Sprintf("docker save %s:%s | gzip | %s docker load",
		config.Image, config.Tag, getSSHCommand(config))
	if _, err := ExecuteCommand(logger, deployCmd, "Transferring Docker image to server"); err != nil {
		return err
	}

	// Copy environment file if it exists
	if _, err := os.Stat(config.EnvFile); err == nil {
		copyEnvCmd := fmt.Sprintf("scp %s %s %s@%s:~/%s",
			getSSHKeyFlag(config), config.EnvFile, config.User, config.Host, config.EnvFile)
		if _, err := ExecuteCommand(logger, copyEnvCmd, "Copying environment file to server"); err != nil {
			return err
		}
	}

	// Prepare remote commands
	envFileFlag := ""
	if _, err := os.Stat(config.EnvFile); err == nil {
		envFileFlag = fmt.Sprintf("--env-file ~/%s", config.EnvFile)
	}

	remoteCommands := strings.Join([]string{
		fmt.Sprintf("docker stop %s || true", config.ContainerName),
		fmt.Sprintf("docker rm %s || true", config.ContainerName),
		fmt.Sprintf("docker run -d --name %s --restart unless-stopped -p %s:%s %s %s:%s",
			config.ContainerName, config.HostPort, config.ContainerPort,
			envFileFlag, config.Image, config.Tag),
	}, " && ")

	// Execute remote commands
	restartCmd := fmt.Sprintf("%s \"%s\"", getSSHCommand(config), remoteCommands)
	if _, err := ExecuteCommand(logger, restartCmd, "Restarting container on server"); err != nil {
		return err
	}

	// Verify container is running
	verifyCmd := fmt.Sprintf("%s \"docker ps --filter name=%s --format '{{.Status}}'\"",
		getSSHCommand(config), config.ContainerName)
	result, err := ExecuteCommand(logger, verifyCmd, "Verifying container status")
	if err != nil {
		return err
	}

	if !strings.Contains(result.Stdout, "Up") {
		return fmt.Errorf("container failed to start properly")
	}

	return logger.Info("Deployment completed successfully! 🚀")
}

func main() {
	logger, err := NewLogger("deploy.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	config := LoadConfig()

	if config.Rollback {
		if err := Rollback(&config, logger); err != nil {
			logger.Error("Rollback failed", err)
			os.Exit(1)
		}
	} else {
		if err := Deploy(&config, logger); err != nil {
			logger.Error("Deployment failed", err)
			os.Exit(1)
		}
	}
}
