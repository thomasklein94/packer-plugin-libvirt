package libvirt

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep"

	libvirt "github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

type communicatorAddressHelper struct {
	InterfaceDef *libvirtxml.DomainInterface
	Source       uint32
}

func GetDomainCommunicatorAddress(state multistep.StateBag) (string, error) {
	config := state.Get("config").(*Config)
	debug := state.Get("debug").(bool)
	driver := state.Get("driver").(*libvirt.Libvirt)
	helper := state.Get("communicator_address_helper").(*communicatorAddressHelper)
	domainRef := state.Get("domain").(*libvirt.Domain)

	host := config.Communicator.Host()
	if host != "" {
		log.Printf("Using user specified host '%s' for communication\n", host)
		return host, nil
	}

	interfaces, err := driver.DomainInterfaceAddresses(*domainRef, helper.Source, 0)
	if err != nil {
		return "", err
	}
	if config.PackerDebug {
		log.Printf("Discovered domain interfaces: %+v\n", interfaces)
	}
	for _, domainIf := range interfaces {
		if len(domainIf.Addrs) == 0 || len(domainIf.Hwaddr) == 0 {
			continue
		}
		for _, hwaddr := range domainIf.Hwaddr {
			if hwaddr != helper.InterfaceDef.MAC.Address || len(domainIf.Addrs) == 0 {
				continue
			}

			// Any address assigned to this interface should be fine to connect
			// By picking randomly, we "give a chanche" to all. If for some reason
			// an address would be unreachable for us, the other might work.
			// An example for this would be trying to connect to a dual-stack host
			// on IPv6 address while the local/bastion machine do not support IPv6 routing.

			addresses := []string{}
			for _, a := range domainIf.Addrs {
				addr, err := formatAddress(a)
				if err != nil {
					if debug {
						log.Println(err)
					}
					continue
				}
				addresses = append(addresses, addr)
			}

			if len(addresses) > 0 {
				randomIndex := rand.Intn(len(addresses))
				pick := addresses[randomIndex]

				return pick, nil
			}
		}
	}

	return "", fmt.Errorf("no suitable IP address found")
}

func formatAddress(domAddr libvirt.DomainIPAddr) (string, error) {
	// In some extreme setups it can happen that packer can only communicate on a link-local interface without DHCP.
	// In those cases, using a link-local IPv6 address might be beneficial, hence this small "workaround".
	// Currently, this is an undocumented feature and use it with extreme caution!
	linkLocalIf := os.Getenv("PACKER_LIBVIRT_COMMUNICATOR_LINK_LOCAL_INTERFACE")

	switch libvirt.IPAddrType(domAddr.Type) {
	case libvirt.IPAddrTypeIpv6:
		// IPv6 Addresses needs to be put into brackets but libvirt API do not present those brackets, so we have to add those here
		addr, _, err := net.ParseCIDR(fmt.Sprintf("%s/%d", domAddr.Addr, domAddr.Prefix))
		if err != nil {
			return "", err
		}
		if addr.IsLinkLocalUnicast() {
			if linkLocalIf == "" {
				return "", fmt.Errorf("unable to use ipv6 link local address '%s' for communication", domAddr.Addr)
			}

			// IPv6 link-local addresses must be formatted as [<address>%<interface>]
			return fmt.Sprintf("[%s%%%s]", domAddr.Addr, linkLocalIf), nil
		}

		return fmt.Sprintf("[%s]", domAddr.Addr), nil

	default:
		return domAddr.Addr, nil
	}
}
