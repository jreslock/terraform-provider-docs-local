FROM golang:1.22-bullseye

# Install basic development tools
RUN apt-get update && apt-get install -y \
    git \
    curl \
    wget \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools
RUN go install github.com/go-task/task/v3/cmd/task@v3.34.1 && \
    go install github.com/caarlos0/svu@v1.9.0 && \
    go install github.com/goreleaser/goreleaser@v1.22.1 && \
    go install golang.org/x/tools/cmd/goimports@v0.17.0 && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2 && \
    pip install pre-commit

# Add Go binaries to PATH
ENV PATH="${PATH}:$(go env GOPATH)/bin"

# Set working directory
WORKDIR /workspace
