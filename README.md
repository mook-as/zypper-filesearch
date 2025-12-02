# zypper-filesearch

This is a zypper plugin that allows searching packages by the file paths they
contain, using a glob pattern.

Usage:

```sh
zypper filesearch [pattern]
```

For example:

```sh
zypper filesearch /usr/lib64/libpng.so
zypper filesearch /usr/lib64/*.so
```
