FROM golang:1.24-bullseye

# Install basic development tools
RUN DEBIAN_FRONTEND=noninteractive apt-get update -yqq && apt-get install -yqq \
    git \
    curl \
    jq \
    wget \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools
RUN go install github.com/go-task/task/v3/cmd/task@v3.42.1 && \
    go install github.com/caarlos0/svu/v3@v3.2.3 && \
    go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest && \
    go install github.com/goreleaser/goreleaser/v2@v2.8.2 && \
    go install golang.org/x/tools/cmd/goimports@v0.31.0 && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.0.2 && \
    pip install -q pre-commit

# Add Go binaries to PATH
ENV PATH="${PATH}:$(go env GOPATH)/bin"

# Set working directory
WORKDIR /workspace
