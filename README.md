# Copepod: Docker Deployment Tool

⚠️ **EXPERIMENTAL PROTOTYPE** - This tool is currently in experimental stage and should be used with caution in production environments.

A simple yet powerful Go-based CLI tool for automating Docker container deployments to remote servers. This tool handles the entire deployment process including building Docker images, transferring them to remote hosts, and managing container lifecycle.

<https://github.com/user-attachments/assets/4034a7b3-67d6-4e9d-88ef-70bd6d1906d7>

## Prerequisites

- Docker installed locally and on the remote host
- SSH access to the remote host
- SSH key-based authentication

## Installation

### Pre-built Binaries

Download the latest pre-built binary from the [releases page](https://github.com/bjarneo/copepod/releases).

Available binaries:

- Linux (AMD64): `copepod-linux-amd64`
- Linux (ARM64): `copepod-linux-arm64`
- macOS (Intel): `copepod-darwin-amd64`
- macOS (Apple Silicon): `copepod-darwin-arm64`

After downloading:

1. Verify the checksum (SHA-256):

```bash
sha256sum -c copepod-<os>-<arch>.sha256
```

2. Make the binary executable:

```bash
chmod +x copepod-<os>-<arch>
```

3. Optionally, move to your PATH:

```bash
# Example for Linux/macOS
sudo mv copepod-<os>-<arch> /usr/local/bin/copepod
```

### Building from Source

Alternatively, you can build from source:

Requirements:

- Go 1.x or higher

```bash
git clone <repository-url>
cd copepod
go build -o copepod
```

## Usage

```bash
./copepod [options]
```

### Command Line Options

| Option           | Environment Variable | Default          | Description                    |
|-----------------|---------------------|------------------|----------------------------------|
| --host          | DEPLOY_HOST       |                  | Remote host to deploy to          |
| --user          | DEPLOY_USER       |                  | SSH user for remote host          |
| --image         | DEPLOY_IMAGE      | copepod_app      | Docker image name                 |
| --tag           | DEPLOY_TAG        | latest           | Docker image tag                  |
| --platform      | DEPLOY_PLATFORM   | linux/amd64      | Docker platform                   |
| --ssh-key       | SSH_KEY_PATH      |                  | Path to SSH key                   |
| --container-name| CONTAINER_NAME    | copepod_app      | Name for the container            |
| --container-port| CONTAINER_PORT    | 3000             | Container port                    |
| --host-port     | HOST_PORT         | 3000             | Host port                         |
| --env-file      | ENV_FILE          |                  | Environment file                  |
| --build-arg     |                   |                  | Build arguments (KEY=VALUE)       |
| --rollback      |                   |                  | Rollback to the previous instance |

### Example Commands

Basic deployment:

```bash
./copepod --host example.com --user deploy
```

Deployment with custom ports:

```bash
./copepod --host example.com --user deploy --container-name myapp --container-port 8080 --host-port 80
```

Using environment file:

```bash
./copepod --env-file .env.production
```

Rollback:

```bash
# For rollback to work you need to deploy using different tags, and not override the same tag each deploy
./codepod --host example.com --user deploy --container-name myapp --container-port 8080 --host-port 80 --rollback
```

Using build arguments:

```bash
# Single build argument
./copepod --host example.com --user deploy --build-arg VERSION=1.0.0

# Multiple build arguments
./copepod --host example.com --user deploy --build-arg VERSION=1.0.0 --build-arg ENV=prod

# Using environment variable
export BUILD_ARGS="VERSION=1.0.0,ENV=prod"
./copepod --host example.com --user deploy

# Using git commit hash
./copepod --host example.com --user deploy --build-arg GIT_HASH=$(git rev-parse HEAD)
```

## Directory Structure

Your project directory should look like this:

```
.
├── Dockerfile            # Required: Docker build instructions
├── your_code            # Your code
└── .env.production      # Optional: Environment variables
```

## Deployment Process

1. Validates configuration and checks prerequisites
2. Verifies Docker installation and SSH connectivity
3. Builds Docker image locally with any provided build arguments
4. Transfers image to remote host
5. Copies environment file (if specified)
6. Stops and removes existing container
7. Starts new container with specified configuration
8. Verifies container is running properly

## Logging

The tool maintains detailed logs in `deploy.log`, including:

- Timestamp for each operation
- Command execution details
- Success/failure status
- Error messages and stack traces

## Error Handling

The tool includes error handling for common scenarios:

- Missing Dockerfile
- SSH connection failures
- Docker build/deployment errors
- Container startup issues

## Security Considerations

- Uses SSH key-based authentication
- Supports custom SSH key paths
- Environment variables can be passed securely via env file
- Build arguments can be used for sensitive build-time variables
- No sensitive information is logged

## Known Limitations

1. Limited error recovery mechanisms
2. Basic container health checking
3. No support for complex Docker network configurations
4. No Docker Compose support

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## Release Process

New versions are automatically built and released when a new tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will trigger the GitHub Action workflow to:

1. Build binaries for multiple platforms
2. Generate checksums
3. Create a new release with the binaries

## TODO

- [ ] Improve error handling
- [ ] Add support for Docker Compose
- [ ] Implement proper container health checks
- [ ] Add shell completion support

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Disclaimer

This is an experimental prototype. Use at your own risk. The authors assume no liability for the use of this tool. Always review the code and test in a safe environment before using in any critical systems.
