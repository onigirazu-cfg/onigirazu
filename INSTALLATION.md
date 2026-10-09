# Installation Guide

## Pre-built Binaries

Download an archive for your platform from [GitHub Releases](https://github.com/onigirazu-cfg/onigirazu/releases).
Archives are named `onigirazu_<Os>_<Arch>.tar.gz` (`.zip` for Windows). See [docs/PLATFORMS.md](docs/PLATFORMS.md) for the full list.

### Linux

```bash
# x86_64
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# ARM64
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_arm64.tar.gz
tar -xzf onigirazu_Linux_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

### macOS

```bash
# Intel
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_x86_64.tar.gz
tar -xzf onigirazu_Darwin_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Apple Silicon
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

### Windows

1. Download `onigirazu_Windows_x86_64.zip` (or `onigirazu_Windows_i386.zip`) from the [releases page](https://github.com/onigirazu-cfg/onigirazu/releases).
2. Extract it and add the directory to `PATH`.

### Verify the download

```bash
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/checksums.txt
sha256sum --ignore-missing -c checksums.txt
```

## Linux Packages

Packages are published for amd64, arm64, armv6, armv7 and 386. File names contain the version
(`onigirazu_<version>_<arch>.<ext>`), so pick the file from the release page, for example:

```bash
VERSION=1.99.0  # example; use the release you want, without the leading "v"
BASE=https://github.com/onigirazu-cfg/onigirazu/releases/download/v${VERSION}

# Debian/Ubuntu
curl -LO $BASE/onigirazu_${VERSION}_amd64.deb
sudo dpkg -i onigirazu_${VERSION}_amd64.deb

# RHEL/Fedora
curl -LO $BASE/onigirazu_${VERSION}_amd64.rpm
sudo rpm -i onigirazu_${VERSION}_amd64.rpm

# Alpine
curl -LO $BASE/onigirazu_${VERSION}_amd64.apk
sudo apk add --allow-untrusted onigirazu_${VERSION}_amd64.apk

# Arch Linux
curl -LO $BASE/onigirazu_${VERSION}_amd64.pkg.tar.zst
sudo pacman -U onigirazu_${VERSION}_amd64.pkg.tar.zst
```

Packages install:

- the binary as `/usr/bin/onigirazu`;
- `/etc/onigirazu/onigirazu.yml`, created on first install from `/usr/share/onigirazu/onigirazu.default.yml` (an existing file is kept);
- example configs in `/usr/share/onigirazu/examples/`;
- documentation in `/usr/share/doc/onigirazu/`.

Packages depend on `git`. Settings: [docs/CONFIGURATION_REFERENCE.md](docs/CONFIGURATION_REFERENCE.md).

## Container Image

Images are published to GitHub Container Registry only, for `linux/amd64` and `linux/arm64`.
Tags: `latest`, `<version>`, `<major>.<minor>`, `<major>`.

```bash
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest --version

docker run --rm -v "$(pwd)":/work -w /work ghcr.io/onigirazu-cfg/onigirazu:latest \
  apply site.yml -i inventory.yml
```

The image is built `FROM scratch`: it contains only the binary, CA certificates and time zone data,
and runs as an unprivileged user.

## Build from Source

Requires Go 1.26.9 or later (see `go.mod`) and Git.

```bash
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu
make build            # writes bin/onigirazu
make install          # copies it to /usr/local/bin (uses sudo)
```

Or with `go install`:

```bash
go install github.com/onigirazu-cfg/onigirazu/cmd/onigirazu@latest
```

## Verify Installation

```bash
onigirazu --version
onigirazu version            # version and the list of modules
onigirazu --help
```

## Updating

- Binaries: replace the binary with the one from the new release.
- Packages: install the new package with the same command.
- Container: `docker pull ghcr.io/onigirazu-cfg/onigirazu:latest`.

## Requirements

Binaries are statically linked (`CGO_ENABLED=0`). SSH is built in; no `ssh` client is needed.
The `git` module needs `git` on the managed host.

## Next Steps

- [Configuration Reference](docs/CONFIGURATION_REFERENCE.md)
- [Module Documentation](docs/modules/README.md)
- [Examples](examples/)
