---
---

## stamp man

Manage stamp troff man pages

### Synopsis

Command group to generate, install, and check stamp man pages.

```
stamp man [flags]
```

### Examples

```
  # install the man page to the default (user) location
  stamp man install

  # check whether the installed man page matches this version
  stamp man check
```

### Options

```
  -h, --help   help for man
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one
* [stamp man check](stamp_man_check.html)	 - Verify installed man page version matches current stamp version
* [stamp man install](stamp_man_install.html)	 - Install the stamp man page to system or user path

