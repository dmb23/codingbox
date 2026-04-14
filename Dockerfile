FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# Base development tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    wget \
    git \
    jq \
    ripgrep \
    fd-find \
    neovim \
    zsh \
    bash \
    ca-certificates \
    gnupg \
    gosu \
    sudo \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Node.js 22 LTS
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# Python tooling via uv
RUN curl -LsSf https://astral.sh/uv/install.sh | sh
ENV PATH="/root/.local/bin:${PATH}"
RUN uv python install 3.14
RUN uv tool install ruff
RUN uv tool install ty

# Go (latest stable)
ARG GO_VERSION=1.24.1
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-$(dpkg --print-architecture).tar.gz" \
    | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# Claude Code
RUN npm install -g @anthropic-ai/claude-code

# OpenCode
RUN curl -fsSL https://opencode.ai/install | bash \
    && cp /root/.opencode/bin/opencode /usr/local/bin/opencode

# Mistral Vibe
RUN curl -LsSf https://mistral.ai/vibe/install.sh | bash

# Entrypoint handles UID/GID matching
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["/bin/bash"]
