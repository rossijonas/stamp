---
---

## Installation

### Quick install (curl)

```bash
curl -fsSL https://gostamp.dev/install | bash
```

### Go install

```bash
go install github.com/rossijonas/stamp/cmd/stamp@latest
```

### GitHub Releases

Download the pre-built binary for your platform from the latest release.

```bash
# Linux x86_64
curl -fsSL https://github.com/rossijonas/stamp/releases/latest/download/stamp_linux_amd64.tar.gz | tar xz
mv stamp ~/.local/bin/

# macOS (Apple Silicon)
curl -fsSL https://github.com/rossijonas/stamp/releases/latest/download/stamp_darwin_arm64.tar.gz | tar xz
mv stamp ~/.local/bin/
```

Replace `linux_amd64` with `darwin_amd64` (Intel Mac) or `linux_arm64` (ARM Linux) as needed.

### From source

```bash
git clone https://github.com/rossijonas/stamp.git
cd stamp
go build -o bin/stamp ./cmd/stamp
cp bin/stamp ~/.local/bin/
```

### PATH Setup

Stamp installs to `~/.local/bin/` by default. If this directory is not in your `$PATH`, add it:

```bash
# Bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc

# Zsh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc

# Fish
fish_add_path "$HOME/.local/bin"
```

### Platform notes

Stamp runs on **Linux** and **macOS**. Windows support is planned.

- **Linux**: Works with DNF, APT, Flatpak, Snap, Zypper, Pacman, and Brew
- **macOS**: Works with Brew and MacPorts
- No additional runtime dependencies — Stamp is a single static binary

### Uninstall

#### Standard (binary only)

```bash
rm $(which stamp)
```

#### Hard (remove all data)

```bash
rm -rf ~/.config/stamp ~/.local/share/stamp
rm -f $(which stamp)
sudo rm -f /usr/local/share/man/man1/stamp.1
```

> **Warning:** Hard uninstall removes your manifest and all snapshots. Back them up first if you plan to restore later:
> ```bash
> cp -r ~/.config/stamp ~/.config/stamp.backup
> cp -r ~/.local/share/stamp ~/.local/share/stamp.backup
> ```
