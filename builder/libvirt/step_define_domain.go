package libvirt

import (
	"context"
	"log"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"libvirt.org/go/libvirtxml"
)

type stepDefineDomain struct{}

func (s *stepDefineDomain) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(*libvirt.Libvirt)
	domainDef := state.Get("domain_def").(*libvirtxml.Domain)

	ui.Say("Sending the domain definition to libvirt")

	xmldesc, err := domainDef.Marshal()

	if err != nil {
		return haltOnError(ui, state, "DefineDomain.Marshal: %s", err)
	}

	if config.PackerDebug {
		log.Printf("domain definition XML:\n%s\n", xmldesc)
	}

	domain, err := driver.DomainDefineXML(xmldesc)

	if err != nil {
		return haltOnError(ui, state, "DefineDomain.RPC: %s", err)
	}

	state.Put("domain", &domain)
	d, err := driver.DomainGetXMLDesc(domain, 0)

	if err == nil {
		domainDef.Unmarshal(d)
	} else {
		log.Printf("couldn't refresh domain definition: %s\n", err)
	}

	var commIface *libvirtxml.DomainInterface = nil

	for _, ni := range domainDef.Devices.Interfaces {
		if ni.Alias != nil && ni.Alias.Name == config.CommunicatorInterface {
			commIface = &ni
		}
	}

	netaddrSource, _ := mapNetworkAddressSources(config.NetworkAddressSource)

	state.Put("communicator_address_helper", &communicatorAddressHelper{
		InterfaceDef: commIface,
		Source:       netaddrSource,
	})

	return multistep.ActionContinue
}

func (s *stepDefineDomain) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)
	domain, ok := state.GetOk("domain")
	driver := state.Get("driver").(*libvirt.Libvirt)

	if ok && domain != nil {
		ui.Say("Undefining the domain...")
		domain := domain.(*libvirt.Domain)
		driver.DomainUndefineFlags(*domain, 0)
	}

	state.Remove("domain")
}
