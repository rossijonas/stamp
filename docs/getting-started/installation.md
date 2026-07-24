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

### From source

```bash
git clone https://github.com/rossijonas/stamp.git
cd stamp
go build -o bin/stamp ./cmd/stamp
sudo cp bin/stamp /usr/local/bin/
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
