packer {
  required_plugins {
    libvirt = {
      version = ">= 0.1.0"
      source = "github.com/thomasklein94/libvirt"
    }
    # sshkey = {
    #   version = ">= 1.0.1"
    #   source = "github.com/ivoronin/sshkey"
    # }
  }
}
