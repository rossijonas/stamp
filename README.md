# Stamp

```txt

                              
         █▄
        ▄██▄      ▄
   ▄██▀█ ██ ▄▀▀█▄ ███▄███▄ ████▄
   ▀███▄ ██ ▄█▀██ ██ ██ ██ ██ ██
  █▄▄██▀▄██▄▀█▄██▄██ ██ ▀█▄████▀
                           ██
                           ▀

```

*<p style="text-align:center;">Track your package installation intent across multiple package managers. Rebuild your environment anywhere.</p>*

---

## Intro

`stamp` is a CLI tool that tracks the software you *intentionally* install across fragmented package managers (`dnf`, `flatpak`, `brew`, etc.). It records your choices into a portable, version-controlled TOML manifest. When you move to a new machine,  run `stamp restore` to recreate your exact environment.

**Current Scope:** The MVP of `stamp` is focused on the Red Hat ecosystem (e.g., Fedora), natively supporting the trio of package managers most commonly used on these systems: `dnf`, `flatpak`, and `brew`.

## The Two Workflows

`stamp` is designed to be flexible. You can use it actively as a unified wrapper, or passively as a safety net.

### Workflow A: Active Management (Recommended)
Use `stamp` as your primary package installer. This guarantees 100% traceability of your intent instantly. It will auto-detect the best package manager or allow you to specify one.

**Search across all managers:**
```bash
stamp search ripgrep
```

**Install and track in one step:**
```bash
stamp install htop              # auto-detects dnf on Fedora
stamp install spotify --via flatpak
```

**Remove and untrack:**
```bash
stamp remove htop
```

### Workflow B: The Passive Observer (The Safety Net)
If you or a script accidentally bypass `stamp` and use native tools directly, `stamp` acts as a safety net to retroactively capture your intent.

**1. Install Normally (Bypassing stamp):**
```bash
sudo dnf install ripgrep
brew install jq
```

**2. Reconcile:**
Run `reconcile` periodically. `stamp` compares your current system against its snapshot, detects the newly installed `ripgrep` and `jq`, and prompts you to add them to your manifest.
```bash
stamp reconcile
```

## Rebuilding Your Environment

When you get a new laptop, clone your dotfiles (containing your `manifest.toml`) and run:

```bash
stamp restore
```
`stamp` will read the manifest and execute the appropriate native install commands concurrently.

To keep everything fresh, run a unified update across all your managers at once:
```bash
stamp update
```

## Adding Notes to Packages
You can annotate why you installed a specific package directly in the CLI, which saves it to your manifest. This is incredibly useful for remembering why you needed an obscure tool 6 months later.

```bash
stamp install lazygit --note "better git TUI than default"
```
Or add a note to an existing tracked package:
```bash
stamp edit lazygit --note "Required for the backend build script"
```

## Roadmap

While `stamp` currently targets Red Hat-based systems, our goal is to become the universal intent tracker across all major Linux distributions and macOS.

### Upcoming Milestones
- **[ ] Debian/Ubuntu Ecosystem Support**
  - Implement native adapters for `apt` and `snap`.
  - Add Debian-specific reconciling rules.
  - ![Debian](https://img.shields.io/badge/Debian-D70A53?style=flat&logo=debian&logoColor=white) ![Ubuntu](https://img.shields.io/badge/Ubuntu-E95420?style=flat&logo=ubuntu&logoColor=white)
- **[ ] Arch Linux Support**
  - Implement native adapters for `pacman`.
- **[ ] Developer Toolchains**
  - Track language-specific global installs (`cargo`, `pipx`, `go install`).

## Architecture & Vision

Read the [Project Vision](docs/VISION.md) to understand the "why" behind the project, or check out the [Technical Specs](docs/SPEC.md) and [Architecture Decisions](docs/decisions/).

## Development

Before starting, ensure you have downloaded the required dependencies:
```bash
go mod tidy
```

- **Check**: `task check` (runs verify, lint, test, and security)
- **Build**: `task build`

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPLv3) - see the [LICENSE](LICENSE) file for details.
