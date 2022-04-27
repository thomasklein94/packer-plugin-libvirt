//go:generate packer-sdc struct-markdown

package libvirt

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/rs/xid"
	"github.com/thomasklein94/packer-plugin-libvirt/builder/libvirt/network"
	"github.com/thomasklein94/packer-plugin-libvirt/builder/libvirt/volume"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	// Communicator configuration
	// See [Packer's documentation](https://www.packer.io/docs/communicators) for more.
	Communicator communicator.Config `mapstructure:"communicator"`

	// The libvirt name of the domain (virtual machine) running your build
	// If not specified, a random name with the prefix `packer-` will be used
	DomainName string `mapstructure:"domain_name" required:"false"`
	// The amount of memory to use when building the VM
	// in megabytes. This defaults to 512 megabytes.
	MemorySize int `mapstructure:"memory" required:"false"`
	// The number of cpus to use when building the VM.
	// The default is `1` CPU.
	CpuCount int `mapstructure:"vcpu" required:"false"`

	// Network interface attachments. See [Network](#network) for more.
	NetworkInterfaces []network.NetworkInterface `mapstructure:"network_interface" required:"false"`
	// The alias of the network interface used for the SSH/WinRM connections
	// See [Communicators and network interfaces](#communicators-and-network-interfaces)
	CommunicatorInterface string `mapstructure:"communicator_interface" required:"false"`

	// See [Volumes](#volumes)
	Volumes []volume.Volume `mapstructure:"volume" required:"false"`
	// The alias of the drive designated to be the artifact. To learn more,
	// see [Volumes](#volumes)
	ArtifactVolumeAlias string `mapstructure:"artifact_volume_alias" required:"false"`

	// See [Graphics and video, headless domains](#graphics-and-video-headless-domains).
	DomainGraphics []DomainGraphic `mapstructure:"graphics" required:"false"`

	// The alias of the network interface designated as the communicator interface.
	// Can be either `agent`, `lease` or `arp`. Default value is `agent`.
	// To learn more, see [Communicators and network interfaces](#communicators-and-network-interfaces)
	// in the builder documentation.
	NetworkAddressSource string `mapstructure:"network_address_source" required:"false"`

	LibvirtURI string `mapstructure:"libvirt_uri" required:"true"`

	ctx interpolate.Context
}

func mapNetworkAddressSources(s string) (sf uint32, err error) {
	switch s {
	case "agent":
		sf = uint32(libvirt.DomainInterfaceAddressesSrcAgent)
	case "lease":
		sf = uint32(libvirt.DomainInterfaceAddressesSrcLease)
	case "arp":
		sf = uint32(libvirt.DomainInterfaceAddressesSrcArp)
	default:
		return 0, fmt.Errorf("unkown NetworkAddressSource: %s", s)
	}

	return sf, nil
}

func (c *Config) Prepare(raws ...interface{}) ([]string, error) {
	err := config.Decode(c, &config.DecodeOpts{
		PluginType:         "libvirt",
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter:  &interpolate.RenderFilter{},
	}, raws...)

	if err != nil {
		return nil, err
	}

	errs := &packersdk.MultiError{}
	warnings := make([]string, 0)

	errs = packersdk.MultiErrorAppend(errs, c.Communicator.Prepare(&c.ctx)...)

	if c.DomainName == "" {
		c.DomainName = fmt.Sprintf("packer-%s", xid.New())
	}

	for i, volumeDef := range c.Volumes {
		volume_warnings, volume_errors := volumeDef.PrepareConfig(&c.ctx, c.DomainName)
		warnings = append(warnings, volume_warnings...)
		errs = packersdk.MultiErrorAppend(errs, volume_errors...)
		c.Volumes[i] = volumeDef
	}

	for i, ni := range c.NetworkInterfaces {
		network_warnings, network_errors := ni.PrepareConfig(&c.ctx)
		warnings = append(warnings, network_warnings...)
		errs = packersdk.MultiErrorAppend(errs, network_errors...)
		c.NetworkInterfaces[i] = ni
	}

	user_defined_communicator_interface := c.CommunicatorInterface
	if c.CommunicatorInterface == "" {
		c.CommunicatorInterface = "communicator"
	}
	// Libvirt only accepts alias with the prefix `ua-`
	c.CommunicatorInterface = fmt.Sprintf("ua-%s", c.CommunicatorInterface)

	if len(c.NetworkInterfaces) == 0 {
		warnings = append(warnings, "No network interface defined")
	} else {
		communicator_interface_found := false
		for _, ni := range c.NetworkInterfaces {
			if ni.Alias == c.CommunicatorInterface {
				communicator_interface_found = true
				break
			}
		}
		if !communicator_interface_found {
			if user_defined_communicator_interface != "" {
				errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("no network_interface found with alias '%s'", user_defined_communicator_interface))
			} else {
				warnings = append(warnings, "using first network interface found as communicator interface")
				ni0 := c.NetworkInterfaces[0]
				ni0.Alias = c.CommunicatorInterface
				c.NetworkInterfaces[0] = ni0
			}
		}
	}

	if len(c.Volumes) == 0 {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("no volume has been specified"))
	} else {
		user_defined_artifact_alias := c.ArtifactVolumeAlias
		if user_defined_artifact_alias == "" {
			c.ArtifactVolumeAlias = "artifact"
		}
		// Libvirt only accepts alias with the prefix `ua-`
		c.ArtifactVolumeAlias = fmt.Sprintf("ua-%s", c.ArtifactVolumeAlias)

		volume_found := false
		for _, vol := range c.Volumes {
			// As a part of the volume.PrepareConfig call above, alias will also be prefixed with `ua-` at this point
			if vol.Alias == c.ArtifactVolumeAlias {
				volume_found = true
				break
			}
		}
		if !volume_found {
			if user_defined_artifact_alias != "" {
				errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("no volume found with alias '%s'", user_defined_artifact_alias))
			} else if len(c.Volumes) == 1 {
				warnings = append(warnings, "Using the only defined volume as an artifact")
				vol0 := c.Volumes[0]
				vol0.Alias = c.ArtifactVolumeAlias
				c.Volumes[0] = vol0
			} else {
				errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("please specify an alias for a volume and set artifact_volume_alias on the builder"))
			}
		}
	}

	if c.NetworkAddressSource == "" {
		c.NetworkAddressSource = "agent"
		warnings = append(warnings, "No network_address_source was specified, defaulting to agent. This might hang your build when there are no qemu agent running on the builder machine")
	}

	if _, err := mapNetworkAddressSources(c.NetworkAddressSource); err != nil {
		errs = packersdk.MultiErrorAppend(errs, err)
	}

	if len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil
}
