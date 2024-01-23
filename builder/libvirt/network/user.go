package network

import (
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type UserNetworkInterface struct {
	Dev string `mapstructure:"dev" required:"false"`
}

func (ni *UserNetworkInterface) PrepareConfig(ctx *interpolate.Context) (warnings []string, errs []error) {
	if ni.Dev == "" {
	}
	return
}

func (ni *UserNetworkInterface) UpdateDomainInterface(domainInterface *libvirtxml.DomainInterface) {
	domainInterface.Source = &libvirtxml.DomainInterfaceSource{
		User: &libvirtxml.DomainInterfaceSourceUser{
		},
	}
}