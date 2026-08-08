---
---

## stamp repo add

Add a third-party repository

```
stamp repo add [name] [url] [flags]
```

### Examples

```
  # add a PPA on Debian/Ubuntu systems
  stamp repo add ppa:git-core/ppa -m apt

  # add a COPR repository on Fedora/RHEL
  stamp repo add petersen/cava -m dnf

  # add a .repo file URL (e.g. Brave or Enpass) on Fedora/RHEL
  stamp repo add brave https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo -m dnf

  # add a repo by URL, deriving the name from the URL
  stamp repo add https://yum.enpass.io/enpass-yum.repo -m dnf

  # add a flatpak remote by URL
  stamp repo add flathub https://dl.flathub.org/repo/flathub.flatpakrepo -m flatpak

  # add a homebrew tap
  stamp repo add homebrew/cask -m brew
```

### Options

```
  -h, --help             help for add
  -m, --manager string   package manager (required)
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp repo](stamp_repo.html)	 - Manage third-party repositories

