package network

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type BridgeNetworkInterface struct {
	// For `type = "bridge" interfaces, the name of the host bridge device which the domain connects to
	Bridge string `mapstructure:"bridge" required:"false"`
}

func (ni *BridgeNetworkInterface) PrepareConfig(ctx *interpolate.Context) (warnings []string, errs []error) {
	errs = []error{}
	if ni.Bridge == "" {
		errs = append(errs, fmt.Errorf("network interfaces with type=bridge must specify the bridge attribute"))
	}
	return
}

func (ni BridgeNetworkInterface) UpdateDomainInterface(domainInterface *libvirtxml.DomainInterface) {
	domainInterface.Source = &libvirtxml.DomainInterfaceSource{
		Bridge: &libvirtxml.DomainInterfaceSourceBridge{
			Bridge: ni.Bridge,
		},
	}
}
