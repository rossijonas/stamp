## stamp man

Generate the stamp man page

### Synopsis

Generate the troff man page for stamp.

By default prints the man page to stdout. Use --install to copy to the system
man page directory so 'man stamp' works.

```
stamp man [flags]
```

### Options

```
  -h, --help            help for man
      --install         install man page to system directory
      --prefix string   install prefix (default: ~/.local)
```

### Options inherited from parent commands

```
      --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.md)	 - Track package installation intent across multiple package managers

