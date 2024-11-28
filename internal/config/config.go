package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the deployment configuration
type Config struct {
	Host          string            `json:"host"`
	User          string            `json:"user"`
	Image         string            `json:"image"`
	Dockerfile    string            `json:"dockerfile"`
	Tag           string            `json:"tag"`
	Platform      string            `json:"platform"`
	SSHKey        string            `json:"sshKey"`
	ContainerName string            `json:"containerName"`
	ContainerPort string            `json:"containerPort"`
	HostPort      string            `json:"hostPort"`
	EnvFile       string            `json:"envFile"`
	Rollback      bool              `json:"rollback"`
	BuildArgs     map[string]string `json:"buildArgs"`
	Network       string            `json:"network"`
	Volumes       []string          `json:"volumes"`
	CPUs          string            `json:"cpus"`
	Memory        string            `json:"memory"`
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

// Load loads configuration from command line flags and environment variables
func Load() Config {
	var config Config
	var showHelp bool
	var showVersion bool
	var buildArgs arrayFlags
	var volumeFlags arrayFlags

	// Initialize BuildArgs map
	config.BuildArgs = make(map[string]string)

	// Define command line flags
	flag.StringVar(&config.Host, "host", getEnv("HOST", ""), "Remote host to deploy to")
	flag.StringVar(&config.User, "user", getEnv("HOST_USER", ""), "SSH user for remote host")
	flag.StringVar(&config.Image, "image", getEnv("DOCKER_IMAGE_NAME", "app"), "Docker image name")
	flag.StringVar(&config.Dockerfile, "dockerfile", "Dockerfile", "Path to the Dockerfile")
	flag.StringVar(&config.Tag, "tag", getEnv("DOCKER_IMAGE_TAG", "latest"), "Docker image tag")
	flag.StringVar(&config.Platform, "platform", getEnv("HOST_PLATFORM", "linux/amd64"), "Docker platform")
	flag.StringVar(&config.SSHKey, "ssh-key", getEnv("SSH_KEY_PATH", ""), "Path to SSH key")
	flag.StringVar(&config.ContainerName, "container-name", getEnv("DOCKER_CONTAINER_NAME", "app"), "Name for the container")
	flag.StringVar(&config.ContainerPort, "container-port", getEnv("DOCKER_CONTAINER_PORT", "3000"), "Container port")
	flag.StringVar(&config.HostPort, "host-port", getEnv("HOST_PORT", "3000"), "Host port")
	flag.StringVar(&config.EnvFile, "env-file", getEnv("DOCKER_CONTAINER_ENV_FILE", ""), "Environment file")
	flag.Var(&buildArgs, "build-arg", "Build argument in KEY=VALUE format (can be specified multiple times)")
	flag.Var(&volumeFlags, "volume", "Volume mount in format 'host:container' (can be specified multiple times)")
	flag.StringVar(&config.Network, "network", getEnv("DOCKER_NETWORK", ""), "Docker network to connect to")
	flag.StringVar(&config.CPUs, "cpus", getEnv("DOCKER_CPUS", ""), "Number of CPUs (e.g., '0.5' or '2')")
	flag.StringVar(&config.Memory, "memory", getEnv("DOCKER_MEMORY", ""), "Memory limit (e.g., '512m' or '2g')")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&config.Rollback, "rollback", false, "Rollback to previous version")
	flag.BoolVar(&showVersion, "version", false, "Show version information")

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

	if showVersion {
		fmt.Printf("pipe version %s\n", version)
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
	if envBuildArgs := os.Getenv("DOCKER_BUILD_ARGS"); envBuildArgs != "" {
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

	// Assign volume flags to config
	config.Volumes = []string(volumeFlags)

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Host == "" || c.User == "" {
		return fmt.Errorf("missing required configuration: host and user must be provided")
	}
	return nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// This will be defined on build time
var version string

const helpText = `
Docker Deployment Tool

Usage:
  pipe [options]

Options:
  --host            Remote host to deploy to
  --user            SSH user for remote host
  --image           Docker image name (default: app)
  --dockerfile      Path to the dockerfile (default: Dockerfile)
  --tag             Docker image tag (default: latest)
  --platform        Docker platform (default: linux/amd64)
  --ssh-key         Path to SSH key (default: "")
  --container-name  Name for the container (default: app)
  --container-port  Container port (default: 3000)
  --host-port       Host port (default: 3000)
  --env-file        Environment file (default: "")
  --build-arg       Build arguments (can be specified multiple times, format: KEY=VALUE)
  --network         Docker network to connect to
  --volume          Volume mount (can be specified multiple times, format: host:container)
  --cpus            Number of CPUs (e.g., '0.5' or '2')
  --memory          Memory limit (e.g., '512m' or '2g')
  --rollback        Rollback to the previous version
  --version         Show version information
  --help            Show this help message

Environment Variables:
  HOST                        Remote host to deploy to
  HOST_USER                   SSH user for remote host
  HOST_PORT                   Host port
  HOST_PLATFORM              Docker platform
  SSH_KEY_PATH               Path to SSH key
  DOCKER_IMAGE_NAME          Docker image name
  DOCKER_IMAGE_TAG           Docker image tag
  DOCKER_CONTAINER_NAME      Name for the container
  DOCKER_CONTAINER_PORT      Container port
  DOCKER_BUILD_ARGS          Build arguments (comma-separated KEY=VALUE pairs)
  DOCKER_CONTAINER_ENV_FILE  Environment file
  DOCKER_NETWORK             Docker network to connect to
  DOCKER_CPUS                Number of CPUs
  DOCKER_MEMORY             Memory limit


Examples:
  pipe --host example.com --user deploy
  pipe --host example.com --user deploy --build-arg VERSION=1.0.0 --build-arg ENV=prod
  pipe --env-file .env.production --build-arg GIT_HASH=$(git rev-parse HEAD)
  pipe --host example.com --user deploy --cpus "0.5" --memory "512m"
  pipe --rollback # Rollback to the previous version
` 