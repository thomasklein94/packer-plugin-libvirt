package network

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type DirectNetworkInterface struct {
	// For `type = "direct" interfaces, the name of the host direct device which the domain connects to
	Dev string `mapstructure:"dev" required:"false"`
	Mode string `mapstructure:"mode" required:"false"`
}

func (ni *DirectNetworkInterface) PrepareConfig(ctx *interpolate.Context) (warnings []string, errs []error) {
	errs = []error{}
	if ni.Dev == "" {
		errs = append(errs, fmt.Errorf("network interfaces with type=direct must specify dev"))
	}
	if ni.Mode == "" {
		errs = append(errs, fmt.Errorf("network interfaces with type=direct must specify mode"))
	}

	return
}

func (ni DirectNetworkInterface) UpdateDomainInterface(domainInterface *libvirtxml.DomainInterface) {
	domainInterface.Source = &libvirtxml.DomainInterfaceSource{
		Direct: &libvirtxml.DomainInterfaceSourceDirect{
			Dev: ni.Dev,
			Mode: ni.Mode,
		},
	}
}
