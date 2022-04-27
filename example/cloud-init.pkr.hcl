# data "sshkey" "install" { }

locals {
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

    user_data_part_1 = {
        packages = [
            "qemu-guest-agent",
        ]
        runcmd = [
            ["systemctl", "enable", "--now", "qemu-guest-agent"],
        ]
        ssh_authorized_keys = [
            # data.sshkey.install.public_key,
        ]
    }

    user_data = format("#cloud-config\n%s", jsonencode(local.user_data_part_1))
}
