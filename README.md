# KBI - Kernel Bundle Image

KBI packages Linux kernels and kernel-dependent artifacts as standard OCI images, enabling independent kernel lifecycle management, explicit compatibility guarantees, and reuse across environments.

```
Kernel (KBI) + RootFS (OCI image)
```

Instead of embedding the kernel inside an OS image, KBI treats the kernel as a first-class, independently versioned artifact.

## Design

### Core Concepts

**Kernel Bundle (KBI Image)** is an OCI image containing kernel artifacts in a canonical layout:

```
/kbi/
  vmlinuz
  initrd                (optional)
  modules/<kver>/
  firmware/             (optional)
  bpf/                  (optional)
```

`/kbi/*` is a distribution namespace, not runtime layout.

**Kernel Build Identity (KBI ID)** is a unique identifier for a kernel build, computed from the kernel's content. Published via OCI annotations and used for module/eBPF compatibility binding.

**Add-ons** are separately packaged OCI artifacts that declare compatibility with a specific KBI ID:

- **ModulePack** - out-of-tree kernel modules (`for_kbi_id = <kbi_id>`)
- **BPF Pack** - compiled eBPF programs with attach metadata and required kfuncs (`for_kbi_id = <kbi_id>`)

Pack builds are rejected unless they declare a target KBI ID, either by resolving a target image with `--for` or by passing `--for-kbi-id` directly.

**Resolver** is the enforcement layer that pulls OCI artifacts, validates signatures, enforces `for_kbi_id` binding, checks compatibility, and produces a resolved kernel view.

### Execution Models

**Bare Metal** - KBI is installed via adapter: `vmlinuz -> /boot/vmlinuz-<kver>`, `modules -> /lib/modules/<kver>`. Bootloader configured normally.

**VM / Image Build** - KBI is baked into disk image, same mapping as bare metal or direct kernel boot via hypervisor.

### Compatibility Model

- Kernel-module compatibility enforced via `kbi_id` matching
- eBPF programs verified against BTF/kfunc availability
- Add-ons must declare compatibility
- Incompatible combinations rejected before boot (fail-fast)

### Security Model

- KBI images signed by kernel authority
- ModulePack/BPF Pack signed by vendors
- Resolver enforces signature policy and compatibility binding
- TODO: kernel module signature verification at pack build and install time
- Optional: measured boot

### Why not existing tools?

**Why not bootc?** bootc installs entire OS images with the kernel embedded inside. Updating the kernel means rebuilding the whole OS image. KBI separates kernel from OS so each has its own lifecycle.

**Why not docker buildx?** buildx builds generic container images from Dockerfiles. It has no understanding of kernel semantics: no KBI ID computation, no per-artifact media types, no compatibility annotations, no validation. You could approximate KBI with a Dockerfile, but nothing enforces the conventions.

**Why not LinuxKit?** LinuxKit builds complete OS images where the kernel is one component. The kernel is not independently reusable or verifiable. KBI makes the kernel a standalone artifact that can be composed with any rootfs.

**Why not just tar + a registry?** You lose typed layers (can't distinguish vmlinuz from initrd without unpacking), deterministic identity (KBI ID), compatibility binding (for_kbi_id), and structured metadata (kver, arch, components). KBI gives you all of this on top of standard OCI.

### Design Principles

> Kernel is a governed resource, not an implicit dependency.

Use KBI when kernel lifecycle must be independent, compatibility must be explicit, kernel reuse is required, or kernel add-ons must be controlled.

## Install

```bash
go install github.com/multikernel/kbi/cmd/kbi@latest
```

Or build from source:

```bash
git clone https://github.com/multikernel/kbi.git
cd kbi
go build -o kbi ./cmd/kbi
```

## Quick Start

### Build a KBI image

```bash
kbi build \
  -k /boot/vmlinuz-$(uname -r) \
  -i /boot/initrd.img-$(uname -r) \
  -m /lib/modules/$(uname -r) \
  -b /sys/kernel/btf/vmlinux \
  -c /boot/config-$(uname -r) \
  --kver $(uname -r) \
  --arch amd64 \
  -t registry.io/org/kernel:$(uname -r)
```

### Inspect a KBI image

```bash
kbi inspect registry.io/org/kernel:6.8.0
```

```
KBI ID:      kbi:sha256:3701209414c63c65...
Kernel:      6.8.0
Arch:        amd64
Components:  vmlinuz,initrd,btf,config,modules
Digest:      sha256:a3fd5977466f1c01...
```

### Push / Pull

```bash
kbi push registry.io/org/kernel:6.8.0
kbi pull registry.io/org/kernel:6.8.0
```

Uses Docker credential helpers (`~/.docker/config.json`) for authentication.

### Install to filesystem

```bash
kbi install registry.io/org/kernel:6.8.0 --dest /
```

Extracts to bare metal layout:

```
/boot/vmlinuz-<kver>
/boot/initrd.img-<kver>
/boot/config-<kver>
/boot/btf-<kver>
/lib/modules/<kver>/
/lib/firmware/
```

### Build a ModulePack

```bash
kbi pack build \
  --type modulepack \
  --for registry.io/org/kernel:6.8.0 \
  -m /path/to/modules/ \
  -t registry.io/org/mydriver:1.0
```

Build when the target KBI image is unavailable locally/remotely, but the KBI ID is already known:

```bash
kbi pack build \
  --type modulepack \
  -m /path/to/modules/ \
  --for-kbi-id kbi:sha256:3701209414c63c65... \
  --arch amd64 \
  -t registry.io/org/mydriver:1.0
```

### Build a BPF Pack

```bash
kbi pack build \
  --type bpfpack \
  --for registry.io/org/kernel:6.8.0 \
  --bpf /path/to/bpf/ \
  -t registry.io/org/mybpf:1.0
```

### Inspect a pack

```bash
kbi pack inspect registry.io/org/mydriver:1.0
```

```
Type:        modulepack
For KBI ID:  kbi:sha256:3701209414c63c65...
For Kernel:  6.8.0
Arch:        amd64
Contents:    mydriver.ko
Digest:      sha256:ef45ab...
```

## Artifacts

Only `vmlinuz` is required. All others are optional:

| Artifact | Flag | Description |
|----------|------|-------------|
| vmlinuz | `-k` | Kernel binary (required) |
| initrd | `-i` | Initial ramdisk |
| modules | `-m` | Kernel modules directory |
| config | `-c` | Kernel .config |
| BTF | `-b` | BPF Type Format data |
| firmware | `--firmware` | Firmware files |

## KBI ID

Each KBI image has a deterministic identity derived from its content:

```
kbi_id = sha256(sort([sha256(vmlinuz), sha256(btf), sha256(config)]))
```

Only identity components (vmlinuz, BTF, config) participate in the hash. Modules, initrd, and firmware are bound to the kernel via `kver`, not the KBI ID. This means swapping out modules or updating the initrd does not change the kernel's identity.

The KBI ID is used for:
- Module compatibility verification (`for_kbi_id` binding)
- eBPF program validation against BTF/kfunc availability
- Add-on ecosystem compatibility

## OCI Format

KBI images are standard OCI images with custom media types per layer:

```
application/vnd.kbi.vmlinuz.v1
application/vnd.kbi.initrd.v1
application/vnd.kbi.modules.v1.tar
application/vnd.kbi.firmware.v1.tar
application/vnd.kbi.kernelconfig.v1
application/vnd.kbi.btf.v1
```

Annotations on the manifest:

```
io.multikernel.kbi.id
io.multikernel.kbi.kver
io.multikernel.kbi.arch
io.multikernel.kbi.components
```

No modifications to the OCI spec. KBI images work with any OCI-compliant registry.

## License

Copyright 2026 Multikernel Technologies, Inc.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
