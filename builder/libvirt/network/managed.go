package network

import (
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type ManagedNetworkInterface struct {
	// If `type = "managed", the name of the Libvirt managed virtual network`. Defaults to the `default` network
	Network string `mapstructure:"network" required:"false"`
}

func (ni *ManagedNetworkInterface) PrepareConfig(ctx *interpolate.Context) (warnings []string, errs []error) {
	if ni.Network == "" {
		ni.Network = "default"
	}

	return
}

func (ni *ManagedNetworkInterface) UpdateDomainInterface(domainInterface *libvirtxml.DomainInterface) {
	domainInterface.Source = &libvirtxml.DomainInterfaceSource{
		Network: &libvirtxml.DomainInterfaceSourceNetwork{
			Network: ni.Network,
		},
	}
}
