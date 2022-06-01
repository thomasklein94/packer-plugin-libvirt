build {
  name = "cloudimage-example"

  source "libvirt.builder" {
    volume {
      alias = "artifact"
      format = "qcow2"

      pool = "base-images"
      name = "ubuntu-22.04-lts"

      source {
        type   = "external"
        urls   = ["https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64-disk-kvm.img"]
      }

      capacity = "1G"

      bus        = "sata"
      target_dev = "sda"
    }

    volume {
      source {
        type = "cloud-init"

        network_config = local.network_config
        user_data      = local.user_data
      }
      capacity = "1M"

      bus        = "sata"
      target_dev = "sdb"
    }
  }
}
