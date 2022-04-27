package libvirt

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

type stepCreateDomain struct{}

func (s *stepCreateDomain) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(*libvirt.Libvirt)
	domainDef := state.Get("domain_def").(*libvirtxml.Domain)

	xmldesc, err := domainDef.Marshal()

	if err != nil {
		return haltOnError(ui, state, "CreateDomain.Marshal: %s", err)
	}

	if config.PackerDebug {
		log.Printf("domain definition XML:\n%s\n", xmldesc)
	}

	connect_uri, _ := driver.ConnectGetUri()
	start_flags := libvirt.DomainNone

	if connect_uri[0:4] != "test" {
		start_flags = libvirt.DomainStartAutodestroy
	}

	domain, err := driver.DomainCreateXML(xmldesc, start_flags)

	if err != nil {
		return haltOnError(ui, state, "CreateDomain.RPC: %s", err)
	}

	state.Put("domain", &domain)
	d, err := driver.DomainGetXMLDesc(domain, 0)

	if err == nil {
		domainDef.Unmarshal(d)
	} else {
		log.Printf("couldn't refresh domain definition: %s\n", err)
	}

	if config.PackerDebug {
		if console_alias := os.Getenv("PACKER_LIBVIRT_STREAM_CONSOLE"); console_alias != "" {
			console_alias = fmt.Sprintf("ua-%s", console_alias)
			go stream_console(driver, domain, console_alias, log.Writer())
		}
	}

	var communicator_interface *libvirtxml.DomainInterface = nil

	for _, ni := range domainDef.Devices.Interfaces {
		if ni.Alias != nil && ni.Alias.Name == config.CommunicatorInterface {
			communicator_interface = &ni
		}
	}

	netaddr_source, _ := mapNetworkAddressSources(config.NetworkAddressSource)

	state.Put("communicator_address_helper", &communicatorAddressHelper{
		InterfaceDef: communicator_interface,
		Source:       netaddr_source,
	})

	return multistep.ActionContinue
}

func (s *stepCreateDomain) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)
	domain, domain_ok := state.GetOk("domain")
	driver := state.Get("driver").(*libvirt.Libvirt)

	if domain_ok && domain != nil {
		domain := domain.(*libvirt.Domain)
		err := driver.DomainDestroy(*domain)
		if err != nil {
			err := fmt.Errorf("error during domain cleanup: %s", err)
			ui.Error(err.Error())
		}
	}
}

func stream_console(driver *libvirt.Libvirt, dom libvirt.Domain, device_alias string, stream io.Writer) error {
	err := driver.DomainOpenConsole(dom, libvirt.OptString{device_alias}, stream, uint32(libvirt.DomainConsoleForce))

	if err != nil {
		return err
	}
	return nil
}
