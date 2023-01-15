//go:generate go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc struct-markdown

package network

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type NetworkInterface struct {
	// [required] Type of attached network interface
	Type string `mapstructure:"type" required:"true"`
	// [optional] If needed, a MAC address can be specified for the network interface.
	// If not specified, Libvirt will generate and assign a random MAC address
	Mac string `mapstructure:"mac" required:"false"`
	// [optional] To help users identifying devices they care about, every device can have an alias which must be unique within the domain.
	// Additionally, the identifier must consist only of the following characters: `[a-zA-Z0-9_-]`.
	Alias string `mapstructure:"alias" required:"false"`
	// [optional] Defines how the interface should be modeled for the domain.
	// Typical values for QEMU and KVM include: `ne2k_isa` `i82551` `i82557b` `i82559er` `ne2k_pci` `pcnet` `rtl8139` `e1000` `virtio`.
	// If nothing is specified, `virtio` will be used as a default.
	Model string `mapstructure:"model" required:"false"`

	Bridge  BridgeNetworkInterface  `mapstructure:",squash"`
	Managed ManagedNetworkInterface `mapstructure:",squash"`
}

func (ni *NetworkInterface) PrepareConfig(ctx *interpolate.Context) (warnings []string, errs []error) {
	warnings = []string{}
	errs = []error{}

	if ni.Model == "" {
		ni.Model = "virtio"
	}

	if ni.Alias != "" {
		ni.Alias = fmt.Sprintf("ua-%s", ni.Alias)
	}

	var w []string
	var e []error
	switch ni.Type {
	case "managed", "network":
		w, e = ni.Managed.PrepareConfig(ctx)
	case "bridge":
		w, e = ni.Bridge.PrepareConfig(ctx)
	default:
		errs = append(errs, fmt.Errorf("unsupported network interface type '%s'", ni.Type))
		return
	}
	warnings = append(warnings, w...)
	errs = append(errs, e...)

	return
}

func (ni NetworkInterface) DomainInterface() *libvirtxml.DomainInterface {
	domainInterface := libvirtxml.DomainInterface{
		Model: &libvirtxml.DomainInterfaceModel{
			Type: ni.Model,
		},
	}
	if ni.Mac != "" {
		domainInterface.MAC = &libvirtxml.DomainInterfaceMAC{
			Address: ni.Mac,
		}
	}
	if ni.Alias != "" {
		domainInterface.Alias = &libvirtxml.DomainAlias{
			Name: ni.Alias,
		}
	}

	switch ni.Type {
	case "bridge":
		ni.Bridge.UpdateDomainInterface(&domainInterface)
	case "managed":
		ni.Managed.UpdateDomainInterface(&domainInterface)
	}

	return &domainInterface
}
