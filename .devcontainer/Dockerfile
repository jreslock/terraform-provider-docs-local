# syntax=docker/dockerfile:1

FROM public.ecr.aws/docker/library/golang:1.24.2-alpine

# Add binaries to PATH
ENV PATH="/root/.local/bin:$(go env GOPATH)/bin:${PATH}"

# Install remaining development tools
RUN apk update && \
    apk add --no-cache \
    bash \
    ca-certificates \
    curl \
    git \
    golangci-lint \
    go-task-task \
    jq \
    pre-commit \
    unzip \
    wget \
    zsh

# Install Go tools
RUN ARCH=$(uname -m) && \
    # goreleaser
    curl -sSL -o goreleaser.apk https://github.com/goreleaser/goreleaser/releases/download/v2.8.2/goreleaser_2.8.2_${ARCH}.apk && \
    apk add --no-cache --allow-untrusted goreleaser.apk && \
    rm -fr goreleaser.apk && \
    ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; elif [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi && \
    # git-chglog
    curl -sSL -o git-chglog.tar.gz https://github.com/git-chglog/git-chglog/releases/download/v0.15.4/git-chglog_0.15.4_linux_${ARCH}.tar.gz && \
    tar xzf git-chglog.tar.gz -C /usr/local/bin git-chglog && \
    chmod +x /usr/local/bin/git-chglog && \
    rm git-chglog.tar.gz && \
    # svu
    curl -sSL -o svu.tar.gz https://github.com/caarlos0/svu/releases/download/v3.2.3/svu_3.2.3_linux_${ARCH}.tar.gz && \
    tar xzf svu.tar.gz -C /usr/local/bin svu && \
    chmod +x /usr/local/bin/svu && \
    rm svu.tar.gz && \
    # goimports
    go install golang.org/x/tools/cmd/goimports@v0.31.0 > /dev/null

# Set working directory
WORKDIR /workspace
