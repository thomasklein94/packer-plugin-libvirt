build {
  name = "backing-store-example"

  source "libvirt.builder" {
    volume {
      alias = "artifact"

      target_dev = "sda"
      bus        = "sata"

      pool = "default"
      name = "custom-image"

      source {
        type = "backing-store"

        pool   = "base-images"
        volume = "ubuntu-20.04-lts"
      }
      capacity = "20G"
    }
  }
}
