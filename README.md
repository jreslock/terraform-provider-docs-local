# terraform-provider-docs-local

A tool for managing local copies of Terraform provider documentation. This tool helps you maintain local copies of Terraform provider documentation for offline access and browsing.

Shout out to and inspiration from @pbnj for creating [terradoc](https://github.com/pbnj/terradoc) which is where I got the idea for this project originally.

## Features

- Clone and maintain local copies of Terraform provider documentation
- Sparse checkout to only download the documentation files
- Support for multiple providers
- Easy updates to keep documentation current
- Generate an index file for easy navigation
- View documentation using `glow` or any markdown viewer

## Requirements

This tool expects providers to have a standard repository structure with a top-level `docs` or `website/docs` directory.

## Installation

### macOS (using Homebrew)

```bash
brew install jreslock/tap/terraform-provider-docs-local
```

### Manual Installation

Download the latest release for your platform from the [releases page](https://github.com/jreslock/terraform-provider-docs-local/releases) and put the binary in `/usr/local/bin` or somewhere on your $PATH, make sure it is exectutable (`chmod +x`).

### Building from Source

```bash
git clone https://github.com/jreslock/terraform-provider-docs-local.git
cd terraform-provider-docs-local
go build
```

## Usage

### Configuration

Create a `providers.yaml` file with your desired providers. You can name the file anything you would like but the utility looks for a `providers.yaml` by default if you prefer less typing. You can optionally specify the desired branch to clone. The default behavior if no branch is specified will auto-detect the HEAD branch from the remote repository.

```yaml
target_dir: terraform-providers  # Optional, defaults to "./terraform-providers"
providers:
  aws:
    repo: hashicorp/terraform-provider-aws
    description: AWS provider for Terraform
    branch: main # Optional, if not set we auto-detect the HEAD branch from the remote
  azurerm:
    repo: hashicorp/terraform-provider-azurerm
    description: Azure provider for Terraform
```

### Commands

```bash
# Clone all providers
terraform-provider-docs-local clone-all

# Clone a specific provider
terraform-provider-docs-local clone-one -p aws

# Clone a specific provider at a desired branch
terraform-provider-docs-local clone-one -p sumologic -b master

# List available providers
terraform-provider-docs-local list

# Update all providers
terraform-provider-docs-local update-all

# Update a specific provider
terraform-provider-docs-local update-one -p aws

# Generate index file
terraform-provider-docs-local index

# Clean up all cloned providers
terraform-provider-docs-local clean
```

### Viewing Documentation

After cloning the providers, you can view the documentation using `glow` from within the directory you specified in your `providers.yaml` (or whatever you set `target_dir` to be):

```bash
# Install glow if you haven't already
brew install glow

# View the index
glow .

# Or navigate directly to a provider's documentation
glow terraform-providers/aws/docs/index.md
```

## Development

### Prerequisites

You can either install the prerequisites locally or use the provided devcontainer:

#### Option 1: Local Development

- [Go](https://golang.org/doc/install) 1.21 or later
- [Task](https://taskfile.dev/installation/) (for running development tasks)
- [svu](https://github.com/caarlos0/svu#installation) (for version management)
- [goreleaser](https://goreleaser.com/install/) (for releases)
- [golangci-lint](https://golangci-lint.run/usage/install/) (for linting)
- [pre-commit](https://pre-commit.com/#install) (for git hooks)
- [testify](https://github.com/stretchr/testify#installation) (for testing)

#### Option 2: Devcontainer (Recommended)

1. Install [Docker](https://docs.docker.com/get-docker/)
2. Install [VS Code](https://code.visualstudio.com/)
3. Install the [Remote - Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension
4. Clone this repository
5. Open the repository in VS Code
6. When prompted, click "Reopen in Container"

The devcontainer includes all required tools and is pre-configured with:

- Go
- Task
- svu
- goreleaser
- golangci-lint
- pre-commit
- glow
- testify
- VS Code extensions for Go development
- Automatic formatting and linting on save

### Development Tasks

```bash
# Build the binary
task build

# Run tests
task test

# Run linters
task lint
```

### Setting up Pre-commit Hooks

Set up the development environment (this will install hooks and run initial checks):

    ```bash
    task pre-commit-install
    ```

Additional pre-commit tasks:

```bash
    # Run pre-commit hooks on all files
    task pre-commit-run

    # Update pre-commit hooks to latest versions
    task pre-commit-update
```

The pre-commit hooks will run the following checks:

- Code formatting (go-fmt)
- Code vetting (go-vet)
- Import formatting (go-imports)
- Linting (golangci-lint)
- Unit tests (go-unit-tests)
- Build verification (go-build)
- Module tidy check (go-mod-tidy)
- General file checks (trailing whitespace, YAML validity, etc.)

## Contributing

1. Create an issue
2. Fork the repository
3. Create your feature branch (`git checkout -b feature/amazing-feature`)
4. Commit your changes (`git commit -m 'Add some amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request and note the issue in the PR title or description

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
