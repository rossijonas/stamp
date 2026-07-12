## stamp completion

Generate shell completion script

### Synopsis

Generate shell completion scripts for stamp.

To load completions:

Bash:

  $ source <(stamp completion bash)

  # To load for each session:
  # Linux:
  $ stamp completion bash > /etc/bash_completion.d/stamp
  # macOS:
  $ stamp completion bash > $(brew --prefix)/etc/bash_completion.d/stamp

Zsh:

  # If completion not enabled, run once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  $ stamp completion zsh > "${fpath[1]}/_stamp"

fish:

  $ stamp completion fish | source

  # To load for each session:
  $ stamp completion fish > ~/.config/fish/completions/stamp.fish

PowerShell:

  PS> stamp completion powershell | Out-String | Invoke-Expression


```
stamp completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.md)	 - A lightweight yet powerful wrapper for your native package managers

