FROM node:22-bookworm

ARG TZ
ENV TZ="$TZ"

ARG CLAUDE_CODE_VERSION=latest
ARG USERNAME=dev

# System packages: shell conveniences and dev tooling. No C/build deps — the
# monitor builds with CGO_ENABLED=0 (ADR-0003), so a C toolchain is unneeded.
RUN apt-get update && apt-get install -y --no-install-recommends \
    less procps sudo fzf zsh man-db unzip jq nano vim \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install GitHub CLI (used by gh git-credential helper and project workflows)
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y gh \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -s /bin/zsh -G sudo $USERNAME \
    && echo "$USERNAME ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Persist shell history
RUN mkdir /commandhistory \
    && touch /commandhistory/.zsh_history \
    && chown -R $USERNAME /commandhistory

# Create workspace and config directories
RUN mkdir -p /workspace /home/$USERNAME/.claude /home/$USERNAME/.codex \
    && chown -R $USERNAME:$USERNAME /workspace /home/$USERNAME/.claude /home/$USERNAME/.codex

# Ensure npm global directory is writable
RUN mkdir -p /usr/local/share/npm-global \
    && chown -R $USERNAME:$USERNAME /usr/local/share/npm-global

ENV DEVCONTAINER=true

# System-level git config survives the bind-mounted ~/.gitconfig from the host
RUN git config --system --add safe.directory /workspace/iscc-monitor

WORKDIR /workspace

# Switch to non-root user for remaining installs
USER $USERNAME
ENV HOME=/home/$USERNAME
ENV NPM_CONFIG_PREFIX=/usr/local/share/npm-global
ENV PATH="$HOME/go/bin:$HOME/.local/bin:/usr/local/share/npm-global/bin:$PATH"
ENV SHELL=/bin/zsh

# Pre-create volume mount-point directories so named volumes inherit dev
# ownership (Go module cache + build cache).
RUN mkdir -p $HOME/go/pkg/mod $HOME/.cache/go-build

# Install mise (tool version manager + task runner). Go itself is provisioned by
# `mise install` at postCreate from mise.toml — keeping mise the single source of
# truth for the Go version (go = "1.24").
RUN curl https://mise.run | sh

# Install Claude Code and Codex CLI
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION} @openai/codex

# Configure ZSH and activate mise in both interactive shells
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended \
    && echo 'export HISTFILE=/commandhistory/.zsh_history' >> $HOME/.zshrc \
    && echo 'eval "$(mise activate zsh)"' >> $HOME/.zshrc \
    && echo 'eval "$(mise activate bash)"' >> $HOME/.bashrc

# Git defaults (overridden at runtime by bind-mounted ~/.gitconfig from host,
# but useful when the container runs without the bind mount)
RUN git config --global init.defaultBranch main \
    && git config --global core.autocrlf input
