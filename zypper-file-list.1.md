# NAME
zypper-file-list - Zypper plugin to list files in an uninstalled package

# SYNOPSIS
**zypper-file-list** [_options_] _packages_

# DESCRIPTION
zypper-file-list is a zypper plugin to list files contained in a package without
having to install it first.

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
List files in the `git` and `libsolv1` packages:
```sh
> zypper file-list git-2.51.0-160000.1.2 libsolv1

Repository       Package   Version            Arch    File
---              ---       ---                ---     ---
repo-oss (16.0)  git       2.51.0-160000.1.2  x86_64  /usr/share/doc/packages/git/README.md
repo-oss (16.0)  libsolv1  0.7.34-160000.2.2  x86_64  /usr/lib64/libsolv.so.1
repo-oss (16.0)  libsolv1  0.7.34-160000.2.2  x86_64  /usr/lib64/libsolvext.so.1
repo-oss (16.0)  libsolv1  0.7.34-160000.2.2  x86_64  /usr/share/licenses/libsolv1/LICENSE.BSD
```
