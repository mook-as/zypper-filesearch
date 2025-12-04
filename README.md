# zypper-filesearch

This is a zypper plugin that allows searching packages by the file paths they
contain, using a glob pattern.

## Usage

```sh
zypper file-search [pattern]
```
[Full zypper file-search documentation](zypper-file-search.1.md)

For example:

```sh
zypper file-search /usr/lib64/libpng.so
zypper file-search /usr/lib64/*.so
```

It is also possible to list files given a package:

```sh
zypper file-list [pattern]
```
[Full zypper file-list documentation](zypper-file-list.1.md)

For example:

```sh
zypper file-list zypper
zypper file-list go-1.24
```

## Installation

This is available on OBS in a [home project]:
```sh
zypper addrepo obs://home:mook_work:golang repo-name
zypper install zypper-filesearch
```

[home project]: https://build.opensuse.org/package/show/home:mook_work:golang/zypper-filesearch
