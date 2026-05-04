# KBI - Kernel Bundle Image

KBI packages Linux kernels and kernel-dependent artifacts as OCI images. It makes the kernel a first-class, independently versioned artifact instead of an implicit part of an OS image.

```text
Bootable system = Kernel artifacts (KBI OCI image) + Root filesystem (standard OCI image)
```

Use KBI when the kernel lifecycle must be independent from user space, when add-on compatibility must be explicit, or when the same kernel needs to be reused across bare-metal, VM, and image-build environments.

## Core Concepts

### Kernel Bundle

A Kernel Bundle is an OCI image containing kernel artifacts in a canonical distribution layout:

```text
/kbi/
  vmlinuz
  initrd                optional
  modules/<kver>/       optional
  firmware/             optional
  bpf/                  optional
```

`/kbi/*` is a distribution namespace, not the runtime filesystem layout.

### KBI ID

The Kernel Build Identity (KBI ID) is a deterministic identifier for a kernel build. It is computed from identity components and published as an OCI annotation:

```text
kbi_id = "kbi:sha256:" + hex(sha256(
  join("\n", sort([
    "vmlinuz:" + hex(sha256(vmlinuz)),
    "btf:"     + hex(sha256(btf)),     // when present
    "config:"  + hex(sha256(config)),  // when present
  ]))
))
```

`vmlinuz` is always included. `btf` and `config` are included only when supplied at build time, so the same `vmlinuz` produces a different KBI ID depending on which identity inputs are present. Modules, initrd, and firmware are bound through metadata such as `kver`, not through the KBI ID. This lets operational payloads change without redefining the kernel build identity.

### Add-On Packs

Add-ons are separately packaged OCI artifacts that bind to a specific KBI ID:

- **ModulePack**: out-of-tree kernel modules
- **BPF Pack**: compiled eBPF objects plus `kbi-bpf.json` metadata

Pack builds are rejected unless they declare a target KBI ID, either by resolving a target image with `--for` or by passing `--for-kbi-id` directly.

### Resolver

The resolver combines one KBI image with zero or more add-on packs and rejects incompatible combinations before boot or install.

Today, the resolver enforces:

- pack `for_kbi_id` matches the KBI ID
- architecture matches
- declared pack kernel version matches when present
- BPF packs require a target KBI with BTF
- BPF packs include manifest metadata

Signature policy, module signature verification, and deep eBPF verification against BTF are planned but not implemented yet.

## Compatibility Model

KBI currently enforces compatibility at these layers:

| Layer | Current enforcement |
|-------|---------------------|
| KBI image | required `vmlinuz`; valid artifact paths; bundled module vermagic must match `--kver` |
| ModulePack | required KBI binding; module vermagic must match target kernel version when known |
| BPF Pack | required KBI binding; required `kbi-bpf.json`; object references must exist; BTF required |
| Resolver | KBI ID, architecture, kernel version, BTF, and manifest checks |

KBI records declared BPF dependencies, including kfuncs and kernel types/fields, but does not yet prove kfunc signatures, CO-RE relocations, or kernel structure compatibility. Those checks belong in a deeper verifier built on the same manifest and KBI BTF.

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

### Build a KBI Image

`--kver` and `--arch` are required because they drive install paths and add-on compatibility checks.

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

### Inspect a KBI Image

```bash
kbi inspect registry.io/org/kernel:6.8.0
```

```text
KBI ID:      kbi:sha256:3701209414c63c65...
Kernel:      6.8.0
Arch:        amd64
Components:  vmlinuz,initrd,btf,config,modules
Digest:      sha256:a3fd5977466f1c01...
```

### Push and Pull

```bash
kbi push registry.io/org/kernel:6.8.0
kbi pull registry.io/org/kernel:6.8.0
```

Registry authentication uses Docker credential helpers from `~/.docker/config.json`.

### Install to a Filesystem

```bash
kbi install registry.io/org/kernel:6.8.0 --dest /
```

The install adapter maps KBI artifacts to the target filesystem:

```text
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

If the target KBI image is unavailable but the KBI ID is known:

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

BPF packs must include `kbi-bpf.json` in the BPF directory, or pass it explicitly with `--bpf-manifest`.

```json
{
  "schema_version": 1,
  "programs": [
    {
      "file": "trace.o",
      "section": "fentry/do_sys_openat2",
      "attach": "fentry",
      "target": "do_sys_openat2"
    }
  ],
  "requires": {
    "btf": true,
    "kfuncs": ["bpf_task_acquire"],
    "kernel_types": [
      {"name": "task_struct", "fields": ["pid", "comm"]}
    ]
  }
}
```

KBI validates that the manifest is present, well-formed, and references object files inside the pack.

### Inspect a Pack

```bash
kbi pack inspect registry.io/org/mydriver:1.0
```

```text
Type:        modulepack
For KBI ID:  kbi:sha256:3701209414c63c65...
For Kernel:  6.8.0
Arch:        amd64
Contents:    mydriver.ko
Digest:      sha256:ef45ab...
```

### Resolve a Kernel View

```bash
kbi resolve \
  registry.io/org/kernel:6.8.0 \
  registry.io/org/mydriver:1.0 \
  registry.io/org/mybpf:1.0
```

`kbi resolve` is a compatibility gate, not a full eBPF verifier. It consumes pack metadata and enforces binding, architecture, kernel version, and BTF presence.

## Artifact Reference

Only `vmlinuz` is required. All other artifacts are optional.

| Artifact | Flag | Notes |
|----------|------|-------|
| vmlinuz | `-k` | kernel binary |
| initrd | `-i` | initial ramdisk |
| modules | `-m` | directory may be `/lib/modules/<kver>` or its parent; module vermagic must match `--kver` |
| config | `-c` | kernel `.config`; participates in KBI ID |
| BTF | `-b` | BPF Type Format data; participates in KBI ID |
| firmware | `--firmware` | firmware directory |

## OCI Format

KBI images are standard OCI images with custom media types per layer:

```text
application/vnd.multikernel.kbi.vmlinuz.v1
application/vnd.multikernel.kbi.initrd.v1
application/vnd.multikernel.kbi.modules.v1.tar
application/vnd.multikernel.kbi.firmware.v1.tar
application/vnd.multikernel.kbi.kernelconfig.v1
application/vnd.multikernel.kbi.btf.v1
```

Pack images use pack-specific layer media types:

```text
application/vnd.multikernel.kbi.modulepack.v1.tar
application/vnd.multikernel.kbi.bpfpack.v1.tar
```

KBI image annotations:

```text
io.multikernel.kbi.id
io.multikernel.kbi.kver
io.multikernel.kbi.arch
io.multikernel.kbi.components
```

Generic pack annotations:

```text
io.multikernel.kbi.pack.type
io.multikernel.kbi.pack.for_kbi_id
io.multikernel.kbi.pack.for_kver
io.multikernel.kbi.pack.contents
io.multikernel.kbi.pack.requires
```

BPF pack annotations:

```text
io.multikernel.kbi.pack.bpf.manifest
io.multikernel.kbi.pack.bpf.programs
io.multikernel.kbi.pack.bpf.kfuncs
io.multikernel.kbi.pack.bpf.types
```

KBI does not modify the OCI specification. KBI images and packs work with OCI-compliant registries.

## Execution Models

**Bare metal:** install KBI artifacts into `/boot`, `/lib/modules/<kver>`, and `/lib/firmware`, then configure the bootloader normally.

**VM or image build:** bake the KBI into a disk image or boot it directly through a hypervisor with an external root filesystem.

## Why KBI?

**Why not bootc?** bootc installs whole OS images with the kernel embedded. KBI separates kernel and rootfs lifecycle.

**Why not docker buildx?** buildx builds generic container images. It does not understand kernel identity, typed kernel layers, module vermagic, BPF metadata, or KBI compatibility binding.

**Why not LinuxKit?** LinuxKit builds complete OS images. KBI makes the kernel a reusable artifact that can be composed with different root filesystems.

**Why not tar plus a registry?** A tarball loses typed layers, deterministic kernel identity, compatibility annotations, structured metadata, and resolver checks.

KBI’s premise is simple: the kernel is a governed resource, not an implicit dependency.

## Security Status

KBI is designed for signed kernel and add-on artifacts, but signature policy enforcement is not implemented yet.

Planned work:

- signature policy enforcement for KBI and pack images
- kernel module signature verification at pack build and install time
- deeper eBPF verification against BTF
- measured boot integration

## License

Copyright 2026 Multikernel Technologies, Inc.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
