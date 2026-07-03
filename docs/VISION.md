# Stamp: Project Vision & Value Proposition

## The Problem: The Workstation Setup Paradox

Modern developers and system administrators use multiple package managers on a single machine. On a typical Fedora or Ubuntu workstation, you might use:
- `dnf` or `apt` for core system packages.
- `flatpak` for sandboxed GUI applications.
- `brew` for the latest CLI utilities.
- `cargo`, `pipx`, or `go install` for language-specific tools.

When it comes time to set up a new machine, restore from a crash, or onboard a new team member, you face the **Workstation Setup Paradox**:
You either have to completely abandon your familiar, native package managers and learn a complex declarative system (like Nix), or you have to manually guess and remember hundreds of individual `install` commands.

Existing tools fail to bridge this gap:
- **Nix / Home Manager:** Incredible reproducibility, but requires a massive philosophical shift, learning a new language, and abandoning `dnf`/`brew`.
- **Ansible:** Industry standard for servers, but too heavy for local workstation intent. It requires you to remember to manually write YAML every time you test a new CLI tool.
- **Topgrade:** Great for *updating* all your managers at once, but it doesn't track *what* you installed.
- **Brewfile:** Perfect, but limited strictly to the Homebrew ecosystem.

## The Solution: Stamp

`stamp` occupies the unfulfilled space in the package management ecosystem: **A multi-manager intention tracking layer that operates without replacing your native tools or changing your habits.**

`stamp` offers the declarative benefits of Nix with the imperative ease of `dnf`. 

### Key Goals & Design Principles

1. **Unified Installation (The Primary Workflow)**
   Use `stamp install <pkg>` as your daily driver. `stamp` auto-detects the best native manager, executes the install, and instantly records your intent. This guarantees 100% traceability from day one.

2. **The Passive Safety Net (`reconcile`)**
   If you or a script accidentally bypass `stamp` and use `dnf install` directly, your intent tracking doesn't break. The `reconcile` command acts as a safety net, detecting the drift and prompting you to track the new package retroactively.
   
3. **Intent vs. State**
   `stamp` does not care about the thousands of dependencies on your system. It only cares about the tools you *intentionally* chose to install. It filters out the noise to create a clean, human-readable manifest.

4. **Multi-Manager by Default**
   A developer's environment spans multiple ecosystems. `stamp` treats `dnf`, `flatpak`, and `brew` as first-class citizens, combining them into a single, unified state file.

5. **Portability & Version Control**
   Your environment is defined by a simple, diff-friendly `manifest.toml`. You commit this file to your dotfiles repository. When you get a new laptop, `git clone` and `stamp restore` brings your entire toolchain back to life in minutes.

6. **Agnostic & Unopinionated**
   `stamp` doesn't dictate how you configure your software (that is the job of `stow` or `chezmoi`). It solely ensures the software *exists* on your machine.

7. **Context Preservation (Notes)**
   Intent is often forgotten. By supporting package-level notes, `stamp` acts as a memory aid. You aren't just restoring `libfoo`, you are restoring `libfoo (required for the legacy billing service compilation)`.

8. **Predictability & Scriptability**
   `stamp` is designed for automation. With the global `--yes` / `-y` flag, it runs deterministically in non-interactive environments (like bootstrapping pipelines), and strictly avoids holding interactive prompts unless stdout is a TTY and the user hasn't explicitly opted into auto-acceptance.

## Who is this for?

`stamp` is built primarily for the **solo developer, SRE, or DevOps engineer** who wants a reproducible local workstation without the cognitive overhead of maintaining local Ansible playbooks or Nix flakes. 

Secondarily, it serves as an ultra-lightweight onboarding tool for small teams: share a `manifest.toml` and guarantee every new hire has the baseline tools required to work.