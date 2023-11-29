The `Libvirt` multi-component plugin can be used with HashiCorp [Packer](https://www.packer.io)
to create custom images. For the full list of available features for this plugin see [docs](docs).

### Installation

To install this plugin, copy and paste this code into your Packer configuration, then run [`packer init`](https://www.packer.io/docs/commands/init).

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


Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
$ packer plugins install github.com/thomasklein94/libvirt
```

### Components

#### Builders

## Builders
- [libvirt](/packer/integrations/thomasklein94/latest/components/builder/libvirt) - The Libvirt Builder is able to create Libvirt volumes on your libvirt hypervisor
  by copying an image from a http source, by using already existing volumes as a backing store,
  or creating an empty one and starting a libvirt domain with these volumes attached.
