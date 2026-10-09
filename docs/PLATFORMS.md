# Supported Platforms

Release builds are defined in [`.goreleaser.yml`](../.goreleaser.yml). All binaries are statically
linked (`CGO_ENABLED=0`) and stripped. For installation commands see [INSTALLATION.md](../INSTALLATION.md).

## Release Archives

Archive name: `onigirazu_<Os>_<Arch>.tar.gz`, `.zip` for Windows.

| OS | Architectures (`<Arch>` in the file name) | Archive example |
|----|-------------------------------------------|-----------------|
| Linux | `x86_64`, `arm64`, `armv6`, `armv7`, `i386` | `onigirazu_Linux_armv7.tar.gz` |
| macOS | `x86_64`, `arm64` | `onigirazu_Darwin_arm64.tar.gz` |
| Windows | `x86_64`, `i386` | `onigirazu_Windows_x86_64.zip` |
| agent for Linux hosts | `amd64`, `arm64`, `armv6`, `386` | embedded in `onigirazu` (`go generate`, run by GoReleaser) |
| agent for Windows hosts | `amd64`, `arm64` | embedded in `onigirazu` |

Every archive also carries the `onigirazu-test` binary (Molecule scenarios without Molecule).
| FreeBSD | `x86_64`, `i386` | `onigirazu_Freebsd_x86_64.tar.gz` |
| OpenBSD | `x86_64`, `i386` | `onigirazu_Openbsd_x86_64.tar.gz` |
| NetBSD | `x86_64`, `i386` | `onigirazu_Netbsd_x86_64.tar.gz` |

There are no universal macOS binaries: use `arm64` on Apple Silicon and `x86_64` on Intel.

Raspberry Pi: `arm64` for a 64-bit OS, `armv7` for Pi 2/3 with a 32-bit OS, `armv6` for Pi 1/Zero.

Each release has `checksums.txt` (SHA-256):

```bash
sha256sum --ignore-missing -c checksums.txt      # Linux
shasum -a 256 --ignore-missing -c checksums.txt  # macOS
```

## Linux Packages

Formats: `.deb`, `.rpm`, `.apk` (Alpine) and `.pkg.tar.zst` (Arch Linux), for `amd64`, `arm64`,
`armv6`, `armv7` and `386`. File name: `onigirazu_<version>_<arch>.<ext>`.

## Container Image

`ghcr.io/onigirazu-cfg/onigirazu`, for `linux/amd64` and `linux/arm64`.

## Minimum OS Versions

Set by the Go toolchain (Go 1.26): Linux kernel 3.2, macOS 12, Windows 10 / Server 2016.

## Other Platforms

Anything else Go supports can be built from source:

```bash
GOOS=linux GOARCH=riscv64 CGO_ENABLED=0 go build -o onigirazu ./cmd/onigirazu
```

Such builds are not produced or tested by CI.
