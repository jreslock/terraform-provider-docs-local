# syntax=docker/dockerfile:1.7

FROM public.ecr.aws/docker/library/golang:1.24.2-bookworm

# Add binaries to PATH
ENV PATH="/root/.local/bin:$(go env GOPATH)/bin:${PATH}"

# Install basic development tools
RUN DEBIAN_FRONTEND=noninteractive apt update -yqq && apt install ca-certificates -yqq && \
    echo 'deb [trusted=yes] https://apt.fury.io/caarlos0/ /' | tee /etc/apt/sources.list.d/caarlos0.list && \
    echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | tee /etc/apt/sources.list.d/goreleaser.list && \
    DEBIAN_FRONTEND=noninteractive apt update -yqq > /dev/null && \
    apt-get install -yqq \
    curl \
    git \
    goreleaser \
    jq \
    pipx \
    svu \
    unzip \
    wget && \
    rm -rf /var/lib/apt/lists/* && \
    pipx install pre-commit && \
    pipx install go-task-bin

# Install Go tools
RUN ARCH=$(dpkg --print-architecture) && \
    # git-chglog
    curl -sSL -o git-chglog.tar.gz https://github.com/git-chglog/git-chglog/releases/download/v0.15.4/git-chglog_0.15.4_linux_${ARCH}.tar.gz && \
    tar xzf git-chglog.tar.gz -C /usr/local/bin git-chglog && \
    chmod +x /usr/local/bin/git-chglog && \
    rm git-chglog.tar.gz && \
    # goimports
    go install golang.org/x/tools/cmd/goimports@v0.31.0 > /dev/null && \
    # golangci-lint
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.0.2 > /dev/null

# Set working directory
WORKDIR /workspace
