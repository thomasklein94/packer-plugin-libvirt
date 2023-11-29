# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "Libvirt"
  description = "Create Libvirt volumes on your libvirt hypervisor."
  identifier = "packer/thomasklein94/libvirt"
  component {
    type = "builder"
    name = "Libvirt"
    slug = "libvirt"
  }
}
