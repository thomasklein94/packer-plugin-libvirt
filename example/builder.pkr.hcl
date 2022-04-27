source "libvirt" "builder" {
  libvirt_uri = "test://./test-libvirt-definition.xml"

  vcpu = 1
  memory = 512

  network_interface {
    type  = "managed"
    alias = "communicator"
  }

  
  communicator {
    communicator = "none"
    ssh_username = "ubuntu"
    # ssh_private_key_file = data.sshkey.install.private_key_path
  }

  network_address_source = "agent"
}
