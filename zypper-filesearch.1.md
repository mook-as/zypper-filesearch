# NAME
zypper-filesearch - Zypper plugin to search for packages by contents

# SYNOPSIS
**zypper-filesearch** [_options_] _terms_

# DESCRIPTION
zypper-filesearch is a zypper plugin to find packages by searching through their
contents without installing them first.  This is normally not required for
executables as zypper searches for files containing the paths `/bin/`, `/sbin/`,
and `/etc/`.

# OPTIONS
**-verbose**
:   Produce extra debug logging.

**-releasever=**_ver_
:   Override the release version; see the same `zypper` option for details.

**-json**
:   Produce output in JSON format.

**-xmlout**
:   Produce output in XML format.

# EXAMPLES
Locate the package providing this package's LICENSE:
```sh
> zypper filesearch '*/zypper-fileseach/LICENSE*'

Repository                 Package            Version         Arch    File
---                        ---                ---             ---     ---
obs:home:mook_work:golang  zypper-filesearch  1.0-lp160.10.1  x86_64  /usr/share/licenses/zypper-filesearch/LICENSE.txt
```
