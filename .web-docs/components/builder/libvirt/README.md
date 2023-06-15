Type: `libvirt`
Artifact BuilderId: `thomasklein94.libvirt`

The Libvirt Builder is able to create Libvirt volumes on your libvirt hypervisor
by copying an image from a http source, by using already existing volumes as a backing store,
or creating an empty one and starting a libvirt domain with these volumes attached.

Libvirt builder also supports cloudinit images during build.

Before looking at the configuration options and examples, you might also want to check out and
familiarize yourself with [libvirt's domain specification](https://libvirt.org/formatdomain.html)
and [libvirt's storage pool and volume concepts](https://libvirt.org/formatstorage.html).

## Installation

To use the pre-built release of Packer Plugin Libvirt, you can add the following snippet to your Packer project:

```hcl
packer {
  required_plugins {
    libvirt = {
      version = ">= 0.5.0"
      source  = "github.com/thomasklein94/libvirt"
    }
  }
}
```

<!-- Builder Configuration Fields -->

## Required

<!-- Code generated from the comments of the Config struct in builder/libvirt/config.go; DO NOT EDIT MANUALLY -->

- `libvirt_uri` (string) - Libvirt URI

<!-- End of code generated from the comments of the Config struct in builder/libvirt/config.go; -->


## Optional

<!-- Code generated from the comments of the Config struct in builder/libvirt/config.go; DO NOT EDIT MANUALLY -->

- `communicator` (communicator.Config) - Communicator configuration
  See [Packer's documentation](https://www.packer.io/docs/communicators) for more.

- `domain_name` (string) - The libvirt name of the domain (virtual machine) running your build
  If not specified, a random name with the prefix `packer-` will be used

- `memory` (int) - The amount of memory to use when building the VM
  in megabytes. This defaults to 512 megabytes.

- `vcpu` (int) - The number of cpus to use when building the VM.
  The default is `1` CPU.

- `cpu_mode` (string) - Set CPU mode, you might want to set it to "host-passthrough".
  See [libvirt documentation](https://libvirt.org/formatdomain.html#cpu-model-and-topology) for more information.
  If not specified, let libvirt decide.

- `network_interface` ([]network.NetworkInterface) - Network interface attachments. See [Network](#network) for more.

- `communicator_interface` (string) - The alias of the network interface used for the SSH/WinRM connections
  See [Communicators and network interfaces](#communicators-and-network-interfaces)

- `volume` ([]volume.Volume) - See [Volumes](#volumes)

- `artifact_volume_alias` (string) - The alias of the drive designated to be the artifact. To learn more,
  see [Volumes](#volumes)

- `boot_devices` ([]string) - Device(s) from which to boot, defaults to hard drive (first volume)
  Available boot devices are: `hd`, `network`, `cdrom`

- `graphics` ([]DomainGraphic) - See [Graphics and video, headless domains](#graphics-and-video-headless-domains).

- `network_address_source` (string) - The alias of the network interface designated as the communicator interface.
  Can be either `agent`, `lease` or `arp`. Default value is `agent`.
  To learn more, see [Communicators and network interfaces](#communicators-and-network-interfaces)
  in the builder documentation.

- `shutdown_mode` (string) - Packer will instruct libvirt to use this mode to properly shut down the virtual machine
  before it attempts to destroy it. Available modes are: `acpi`, `guest`, `initctl`, `signal` and `paravirt`.
  If not set, libvirt will choose the method of shutdown it considers the best.

- `shutdown_timeout` (duration string | ex: "1h5m2s") - After succesfull provisioning, Packer will wait this long for the virtual machine to gracefully
  stop before it destroys it. If not specified, Packer will wait for 5 minutes.

- `domain_type` (string) - [Expert] Domain type. It specifies the hypervisor used for running the domain.
  The allowed values are driver specific, but include "xen", "kvm", "hvf", "qemu" and "lxc".
  Default is kvm.
  If unsure, leave it empty.

- `arch` (string) - [Expert] Domain architecture. Default is x86_64

- `chipset` (string) - Libvirt Machine Type
  Value for domain XML's machine type. If unsure, leave it empty

- `loader_path` (string) - [Expert] Refers to a firmware blob, which is specified by absolute path, used to assist the domain creation process.
  If unsure, leave it empty.

- `loader_type` (string) - [Expert] Accepts values rom and pflash. It tells the hypervisor where in the guest memory the loader(rom) should be mapped.
  If unsure, leave it empty.

- `secure_boot` (bool) - [Expert] Some firmwares may implement the Secure boot feature.
  This attribute can be used to tell the hypervisor that the firmware is capable of Secure Boot feature.
  It cannot be used to enable or disable the feature itself in the firmware.
  If unsure, leave it empty.

- `nvram_path` (string) - [Expert] Some UEFI firmwares may want to use a non-volatile memory to store some variables.
  In the host, this is represented as a file and the absolute path to the file is stored in this element.
  If unsure, leave it empty.`

- `nvram_template` (string) - [Expert] If the domain is an UEFI domain, libvirt copies a so called master NVRAM store file defined in qemu.conf.
  If needed, the template attribute can be used to per domain override map of master NVRAM stores from the config file.
  If unsure, leave it empty.

<!-- End of code generated from the comments of the Config struct in builder/libvirt/config.go; -->


## Aliases
Libvirt offers a way to add one alias per device. Libvirt builder uses this to find out which volume should be the
artifact in case there are multiple volume configurations present. Finding the communicator interface is similar in fashion.
There are reasonable defaults built into the builder to ease onboarding but they will produce a warning and it's 
recommended to use aliases.

The identifier must consist only of the following characters: `[a-zA-Z0-9_-]`. Libvirt also requires the identifier 
to start with `ua-`, but libvirt builder will take care of that for you and prepend every alias with `ua-` before 
applying the domain definition to libvirt.

## Graceful shutdown
To ensure all changes were synced to the artifact volume,
after a **successful provisioning**, the Libvirt plugin will try to gracefully terminate the builder instance
by sending a shutdown command to libvirt and wait up to `shutdown_timeout` before forcefully destroys the domain.
Libvirt supports multiple way to shut down a domain, which can be controlled by the `shutdown_mode` attribute.

## Volumes

Libvirt uses volumes to attach as disks, to boot from and to persist data to. Libvirt Builder treats volumes as sources
and possible artifacts. Arbitrary number of volumes can be attached to a builder domain with specifying a `volume { }` 
block for each. The volume which will be the artifact of the build has to be marked with `alias = "artifact"`.

A volume defined with a source MUST NOT EXISTS BEFORE the build and WILL BE DESTROYED at the end of the build.
The only exception is when the volume marked as an artifact in which case a successful build prevents libvirt builder from 
deleting the volume.

If a volume does not have a source defined and does not marked as an artifact,
the volume must exists before the build, and will not be destroyed at the end of the build.

<!-- Code generated from the comments of the Volume struct in builder/libvirt/volume/volume.go; DO NOT EDIT MANUALLY -->

- `pool` (string) - Specifies the name of the storage pool (managed by libvirt) where the disk resides. If not specified
  the pool named `default` will be used

- `name` (string) - Providing a name for the volume which is unique to the pool. (For a disk pool, the name must be
  combination of the source device path device and next partition number to be
  created. For example, if the source device path is /dev/sdb and there are no
  partitions on the disk, then the name must be sdb1 with the next name being
  sdb2 and so on.)
  If not provided, a name will be generated in a format of `<domain_name>-<postfix>` where
  postfix will be either the alias or a random id.
  For cloudinit, the generated name will be `<domain_name>-cloudinit`

- `source` (\*VolumeSource) - Provides information about the underlying storage allocation of the volume.
  This may not be available for some pool types.

- `size` (string) - Providing the total storage allocation for the volume.
  This may be smaller than the logical capacity if the volume is sparsely allocated.
  It may also be larger than the logical capacity if the volume has substantial metadata overhead.
  If omitted when creating a volume, the volume will be fully allocated at time of creation.
  If set to a value smaller than the capacity, the pool has the option of deciding to sparsely allocate a volume.
  It does not have to honour requests for sparse allocation though.
  Different types of pools may treat sparse volumes differently.
  For example, the logical pool will not automatically expand volume's allocation when it gets full;

- `capacity` (string) - Providing the logical capacity for the volume. This value is in bytes by default,
  but a unit attribute can be specified with the same semantics as for allocation
  This is compulsory when creating a volume.

- `readonly` (bool) - If true, it indicates the device cannot be modified by the guest.

- `target_dev` (string) - DEPRECATED!
  The device order on the bus is determined automatically from the order of definition

- `bus` (string) - The optional bus attribute specifies the type of disk device to emulate;
  possible values are driver specific, with typical values being
  `ide`, `scsi`, `virtio`, `xen`, `usb`, `sata`, or `sd` `sd` since 1.1.2.
  If omitted, the bus type is inferred from the style of the device name
  (e.g. a device named 'sda' will typically be exported using a SCSI bus).

- `alias` (string) - To help users identifying devices they care about, every device can have an alias which must be unique within the domain.
  Additionally, the identifier must consist only of the following characters: `[a-zA-Z0-9_-]`.

- `format` (string) - Specifies the volume format type, like `qcow`, `qcow2`, `vmdk`, `raw`. If omitted, the storage pool's default format
  will be used.

- `device` (string) - Specifies the device type. If omitted, defaults to "disk". Can be `disk`, `floppy`, `cdrom` or `lun`.

<!-- End of code generated from the comments of the Volume struct in builder/libvirt/volume/volume.go; -->


## Backing-store volume source
Backing-store source instructs libvirt to use an already presented volume as a base for this volume.

<!-- Code generated from the comments of the BackingStoreVolumeSource struct in builder/libvirt/volume/backing_store.go; DO NOT EDIT MANUALLY -->

- `pool` (string) - Specifies the name of the storage pool (managed by libvirt) where the disk source resides

- `volume` (string) - Specifies the name of storage volume (managed by libvirt) used as the disk source.

- `path` (string) - The file backing the volume. Mutually exclusive with pool and volume args!

<!-- End of code generated from the comments of the BackingStoreVolumeSource struct in builder/libvirt/volume/backing_store.go; -->


Example

```hcl
volume {
  target_dev = "sda"
  bus        = "sata"

  pool = "default"
  name = "custom-image"

  source {
    type = "backing-store"

    pool   = "base-images"
    volume = "ubuntu-22.04-lts"
  }
  capacity = "20G"
}
```

## Cloning a volume
If you wish to clone a volume instead of using a backing store overlay described above,
You have the option to use the `cloning` source type.

<!-- Code generated from the comments of the CloningVolumeSource struct in builder/libvirt/volume/cloning.go; DO NOT EDIT MANUALLY -->

- `volume` (string) - Specifies the name of storage volume (managed by libvirt) used as the disk source.

<!-- End of code generated from the comments of the CloningVolumeSource struct in builder/libvirt/volume/cloning.go; -->


Example

```hcl
volume {
  target_dev = "sda"
  bus        = "sata"

  pool = "default"
  name = "custom-image"

  source {
    type = "cloning"

    pool   = "base-images"
    volume = "ubuntu-22.04-lts"
  }
}
```

## Cloud-init volume source
Linux domains running in a libvirt environment can be set up with the 
[NoCloud cloud-init datasource](https://cloudinit.readthedocs.io/en/latest/topics/data-source/nocloud.html).
This means a small ISO image with the label `CIDATA` will be assembled at the machine running packer and will be uploaded to the volume.
To assemble an ISO image, one of the following commands must be installed at the machine running packer and must be available 
in the `PATH`: `xorriso`, `mkisofs`, `hdiutil` or `oscdimg`.

For cloud-init backed sources, the size and capacity attributes can be omitted for the volume.

<!-- Code generated from the comments of the CloudInitSource struct in builder/libvirt/volume/cloudinit.go; DO NOT EDIT MANUALLY -->

- `meta_data` (\*string) - [optional] NoCloud instance metadata as string.
  If not present, a default one will be generated with the instance-id set to the same value as the domain name
  If you want to set network config, see the `network_config` attribute

- `user_data` (\*string) - [optional] CloudInit user-data as string.
  See [cloud-init documentation](https://cloudinit.readthedocs.io/en/latest/topics/format.html) for more

- `network_config` (\*string) - [optional] Network configuration for cloud-init to be picked up.
  User-data cannot change an instance’s network configuration.
  In the absence of network configuration in any sources,
  Cloud-init will write out a network configuration that will issue a DHCP request on a “first” network interface.
  Read more about the possible format and features at
  [cloud-init documentation](https://cloudinit.readthedocs.io/en/latest/topics/network-config.html).

<!-- End of code generated from the comments of the CloudInitSource struct in builder/libvirt/volume/cloudinit.go; -->


Example:

```hcl
volume {
  target_dev = "sdb"
  bus        = "sata"
  source {
    type = "cloud-init"

    meta_data = jsonencode({
      "instance-id" = "i-abcdefghijklm"
      "hostname"    = "my-packer-builder"
    })

    user_data =  format("#cloud-config\n%s", jsonencode({
      packages = [
            "qemu-guest-agent",
        ]
        runcmd = [
            ["systemctl", "enable", "--now", "qemu-guest-agent"],
        ]
        ssh_authorized_keys = [
            data.sshkey.install.public_key,
        ]
    }))

    network_config = jsonencode({
      version = 2
        ethernets = {
            eth = {
                match = {
                    name = "en*"
                }
                dhcp4 = true
            }
        }
    })
  }
}
```

## External volume source

<!-- Code generated from the comments of the ExternalVolumeSource struct in builder/libvirt/volume/external.go; DO NOT EDIT MANUALLY -->

An external volume source defines a volume source that is external (or remote) to
the libvirt hypervisor. This usually means a cloud image or installation media
from the web or local machine. It's using go-getter under the hood with support
for HTTP, HTTPS, S3, SMB and local files. The files will be cached by Packer.

<!-- End of code generated from the comments of the ExternalVolumeSource struct in builder/libvirt/volume/external.go; -->

<!-- Code generated from the comments of the ExternalVolumeSource struct in builder/libvirt/volume/external.go; DO NOT EDIT MANUALLY -->

- `checksum` (string) - The checksum and the type of the checksum for the download

- `urls` ([]string) - A list of URLs from where this volume can be obtained

<!-- End of code generated from the comments of the ExternalVolumeSource struct in builder/libvirt/volume/external.go; -->


Example:

```hcl
volume {
  alias = "artifact"

  pool = "base-images"
  name = "ubuntu-22.04-lts"

  source {
    type     = "external"
    urls     = ["https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64-disk-kvm.img"]
    checksum = "2a0d8745dee674a7f3038d627d39ff68e5d7276f97866a4abd9ebfcb3df5fa05"
  }

  format = "qcow2"

  capacity = "1G"

  bus        = "sata"
  target_dev = "sda"
}

```

## Files volume source

<!-- Code generated from the comments of the FilesVolumeSource struct in builder/libvirt/volume/files.go; DO NOT EDIT MANUALLY -->

Create an image with files local to where packer runs and make it available at boot time.
Depending on the parent volume's `device` attribute, the result will be a cdrom or a floppy image.
Use this volume source scarcely, for example when the files needed as early as boot time.
In general, making local files available on the remote machine is better done with the file provisioner.
Floppies must not exceed the size 1.44MiB.

<!-- End of code generated from the comments of the FilesVolumeSource struct in builder/libvirt/volume/files.go; -->

<!-- Code generated from the comments of the FilesVolumeSource struct in builder/libvirt/volume/files.go; DO NOT EDIT MANUALLY -->

- `files` ([]string) - Files can be either files or directories. An files provided here will
  be written to the root of the CD or Floppy. Directories will be written to the
  root of the CD as well, but will retain their subdirectory structure.
  For a cdrom, the behaviour is the same as `cd_files`. For floppy disks, the behaviour
  is the same as for `floppy_dirs`!

- `contents` (map[string]string) - Key/Value pairs to add to the CD or Floppy. The keys represent the paths, and the values
  represent the contents. Can be used together with `files`. If any paths are specified by
  both, `contents` will take precedence.

- `label` (string) - Label

<!-- End of code generated from the comments of the FilesVolumeSource struct in builder/libvirt/volume/files.go; -->


Examples:

* Create a floppy disk for unattended Windows installation *
```hcl
volume {
  alias = "unattend"

  device = "floppy"

  source {
    type = "files"
    label = "unattend"
    contents = {
      "Unattend.xml" = templatefile("${path.root}/Unattend.xml.tmpl", {ComputerName = "MyMachine"})
    }
  }
}
```

* Create a CDROM for drivers stored locally
```hcl
volume {
  alias = "drivers"
  device = "cdrom"

  source {
    type = "files"
    label = "drivers"
    files = ["./drivers/*"]
  }
}
```

## Network
Network interfaces can be attached to a builder domain by adding a `network_interface { }` block for each.
Currently only `managed` and `bridge` networks are supported.

<!-- Code generated from the comments of the NetworkInterface struct in builder/libvirt/network/network.go; DO NOT EDIT MANUALLY -->

- `type` (string) - [required] Type of attached network interface

<!-- End of code generated from the comments of the NetworkInterface struct in builder/libvirt/network/network.go; -->

<!-- Code generated from the comments of the NetworkInterface struct in builder/libvirt/network/network.go; DO NOT EDIT MANUALLY -->

- `mac` (string) - [optional] If needed, a MAC address can be specified for the network interface.
  If not specified, Libvirt will generate and assign a random MAC address

- `alias` (string) - [optional] To help users identifying devices they care about, every device can have an alias which must be unique within the domain.
  Additionally, the identifier must consist only of the following characters: `[a-zA-Z0-9_-]`.

- `model` (string) - [optional] Defines how the interface should be modeled for the domain.
  Typical values for QEMU and KVM include: `ne2k_isa` `i82551` `i82557b` `i82559er` `ne2k_pci` `pcnet` `rtl8139` `e1000` `virtio`.
  If nothing is specified, `virtio` will be used as a default.

<!-- End of code generated from the comments of the NetworkInterface struct in builder/libvirt/network/network.go; -->


## Communicators and network interfaces
If an SSH or WinRM address is specified in the communicator block, packer will use that address to communicate with the
builder domain.

If no SSH or WinRM address is specified, the libvirt builder tries to find an address for initiating communications.
If only one network interface is specified for a builder domain, and there is no `communicator_interface` specified for
the domain, then that interface will be used as the interface for communication.

If more than one network interface is specified, the communicator interface must be tagged by setting the same alias as 
specified with the `communicator_interface` domain attribute (which is "communicator" by default.)

The `network_address_source` domain attribute controls how the address discovery will take place. There are three 
possible way to discover a domain's virtual address:
- `agent`: this is the most reliable method for getting an address for the domain, but it requires `qemu-guest-agent`
           running in the domain. This is the default method for domains.
- `lease`: if for some reason, the guest agent can not be started, but the communicator interface is connected to
           one of libvirt's managed networks, you can use `lease` to see what DHCP lease was offered for the interface.
- `arp`: this method relies on the libvirt host ARP table to find an IP address associated with the MAC address given to
         the domain.

Examples

For connecting to a managed network
```hcl
network_interface {
  type    = "managed"
  network = "my-managed-network"
}
```

For connecting to a bridge
```hcl
network_interface {
  type   = "bridge"
  bridge = "br0"
}
```

## Graphics and video, headless domains
Libvirt builder creates a headless domain by default with no video card or monitor attached to it. Most linux distributions
and cloud images are fine with this setup, but you might need to add a video card and a graphical interface to your machine.

You can add one (or more) graphic device to your machine by using the `graphics { }` block.
If at least one graphic device is added to the builder configuration, a video device with the model `virtio` will
automatically be added to the domain.
Currently, there is no option to specify or customize just a video device for a domain.

<!-- Code generated from the comments of the DomainGraphic struct in builder/libvirt/config_graphics.go; DO NOT EDIT MANUALLY -->

- `type` (string) - Type of the graphic defined with this block. Required. Must be either `vnc` or `sdl`.

<!-- End of code generated from the comments of the DomainGraphic struct in builder/libvirt/config_graphics.go; -->

<!-- Code generated from the comments of the SdlDomainGraphic struct in builder/libvirt/config_graphics.go; DO NOT EDIT MANUALLY -->

- `display` (string) - An X11 Display number where the SDL window will be sent. Required for type `sdl`

<!-- End of code generated from the comments of the SdlDomainGraphic struct in builder/libvirt/config_graphics.go; -->

<!-- Code generated from the comments of the VNCDomainGraphic struct in builder/libvirt/config_graphics.go; DO NOT EDIT MANUALLY -->

- `port` (int) - TCP port used for VNC server to listen on.
  The number zero means the hypervisor will pick a free port randomly.
  If not set, the autoport feature will be used.

<!-- End of code generated from the comments of the VNCDomainGraphic struct in builder/libvirt/config_graphics.go; -->


## Debugging a build
By default, Libvirt builder assigns two serial console to the domain with the aliases `serial-console` and `virtual-console`.
You can use your virtual manager to connect to one of these consoles for debug.

If you don't have access to a virtual machine manager, if specifying the `-debug` flag and `PACKER_LOG=1` environment
variable to packer simultaneously while setting the `PACKER_LIBVIRT_STREAM_CONSOLE` to one of the console aliases, 
the builder will connect to that console and logs any message the domain sends to that console.
In no way, shape or form should this be used in any production or serious setup!

## A simple builder definition example

The following example will download and then upload an ubuntu cloud image to the default storage pool, then starts a
libvirt domain accessing the default libvirt network. Additionally, the example will create an SSH key and add 
a cloud-init image to the domain in order to enable Packer to connect to the domain and run provisioners.

```hcl
packer {
  required_plugins {
    sshkey = {
      version = ">= 1.0.1"
      source = "github.com/ivoronin/sshkey"
    }
    libvirt = {
      version = ">= 0.5.0"
      source  = "github.com/thomasklein94/libvirt"
    }
  }
}

data "sshkey" "install" {
}

source "libvirt" "example" {
  libvirt_uri = "qemu:///system"

  vcpu   = 1
  memory = 846

  network_interface {
    type  = "managed"
    alias = "communicator"
  }

  # https://developer.hashicorp.com/packer/plugins/builder/libvirt#communicators-and-network-interfaces
  communicator {
    communicator         = "ssh"
    ssh_username         = "ubuntu"
    ssh_private_key_file = data.sshkey.install.private_key_path
  }
  network_address_source = "lease"

  volume {
    alias = "artifact"

    source {
      type     = "external"
      # With newer releases, the URL and the checksum can change.
      urls     = ["https://cloud-images.ubuntu.com/releases/22.04/release-20230302/ubuntu-22.04-server-cloudimg-amd64-disk-kvm.img"]
      checksum = "3b11d66d8211a8c48ed9a727b9a74180ac11cd8118d4f7f25fc7d1e4a148eddc"
    }

    capacity   = "10G"
    target_dev = "sda"
    bus        = "sata"
    format     = "qcow2"
  }

  volume {
    source {
      type = "cloud-init"
      user_data = format("#cloud-config\n%s", jsonencode({
        ssh_authorized_keys = [
          data.sshkey.install.public_key,
        ]
      }))

    }

    target_dev = "sdb"
    bus        = "sata"
  }
  shutdown_mode = "acpi"
}

build {
  sources = ["source.libvirt.example"]
  provisioner "shell" {
    inline = [
      "echo The domain has started and became accessible",
      "echo The domain has the following addresses",
      "ip -br a",
      "echo if you want to connect via SSH use the following key: ${data.sshkey.install.private_key_path}",
    ]
  }
  provisioner "breakpoint" {
    note = "You can examine the created domain with virt-manager, virsh or via SSH"
  }
}
```
