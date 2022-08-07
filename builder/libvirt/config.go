//go:generate packer-sdc struct-markdown

package libvirt

import (
	"fmt"
	"time"

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
	//
	BootConfig BootConfig `mapstructure:",squash"`
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

	// Device(s) from which to boot, defaults to hard drive (first volume)
	// Available boot devices are: `hd`, `network`, `cdrom`
	BootDevices []string `mapstructure:"boot_devices" required:"false"`

	// See [Graphics and video, headless domains](#graphics-and-video-headless-domains).
	DomainGraphics []DomainGraphic `mapstructure:"graphics" required:"false"`

	// The alias of the network interface designated as the communicator interface.
	// Can be either `agent`, `lease` or `arp`. Default value is `agent`.
	// To learn more, see [Communicators and network interfaces](#communicators-and-network-interfaces)
	// in the builder documentation.
	NetworkAddressSource string `mapstructure:"network_address_source" required:"false"`

	LibvirtURI string `mapstructure:"libvirt_uri" required:"true"`
	// Packer will instruct libvirt to use this mode to properly shut down the virtual machine
	// before it attempts to destroy it. Available modes are: `acpi`, `guest`, `initctl`, `signal` and `paravirt`.
	// If not set, libvirt will choose the method of shutdown it considers the best.
	ShutdownMode string `mapstructure:"shutdown_mode" required:"false"`
	// After succesfull provisioning, Packer will wait this long for the virtual machine to gracefully
	// stop before it destroys it. If not specified, Packer will wait for 5 minutes.
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" required:"false"`

	// [Expert] Domain type. It specifies the hypervisor used for running the domain.
	// The allowed values are driver specific, but include "xen", "kvm", "hvf", "qemu" and "lxc".
	// Default is kvm.
	// If unsure, leave it empty.
	DomainType string `mapstructure:"domain_type" required:"false"`
	// [Expert] Domain architecture. Default is x86_64
	Arch string `mapstructure:"arch" required:"false"`
	// Libvirt Machine Type
	// Value for domain XML's machine type. If unsure, leave it empty
	Chipset string `mapstructure:"chipset" required:"false"`
	// [Expert] Refers to a firmware blob, which is specified by absolute path, used to assist the domain creation process.
	// If unsure, leave it empty.
	LoaderPath string `mapstructure:"loader_path" required:"false"`
	// [Expert] Accepts values rom and pflash. It tells the hypervisor where in the guest memory the loader(rom) should be mapped.
	// If unsure, leave it empty.
	LoaderType string `mapstructure:"loader_type" required:"false"`
	// [Expert] Some firmwares may implement the Secure boot feature.
	// This attribute can be used to tell the hypervisor that the firmware is capable of Secure Boot feature.
	// It cannot be used to enable or disable the feature itself in the firmware.
	// If unsure, leave it empty.
	SecureBoot bool `mapstructure:"secure_boot" required:"false"`
	// [Expert] Some UEFI firmwares may want to use a non-volatile memory to store some variables.
	// In the host, this is represented as a file and the absolute path to the file is stored in this element.
	// If unsure, leave it empty.`
	NvramPath string `mapstructure:"nvram_path" required:"false"`
	// [Expert] If the domain is an UEFI domain, libvirt copies a so called master NVRAM store file defined in qemu.conf.
	// If needed, the template attribute can be used to per domain override map of master NVRAM stores from the config file.
	// If unsure, leave it empty.
	NvramTemplate string `mapstructure:"nvram_template" required:"false"`

	ctx interpolate.Context
}

func mapDomainShutdown(str string) (result libvirt.DomainShutdownFlagValues, ok bool) {
	var domainShutdownFlagMaps = map[string]libvirt.DomainShutdownFlagValues{
		"auto":     libvirt.DomainShutdownDefault,
		"acpi":     libvirt.DomainShutdownAcpiPowerBtn,
		"guest":    libvirt.DomainShutdownGuestAgent,
		"initctl":  libvirt.DomainShutdownInitctl,
		"signal":   libvirt.DomainShutdownSignal,
		"paravirt": libvirt.DomainShutdownParavirt,
	}

	result, ok = domainShutdownFlagMaps[str]
	return
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

	if len(c.BootDevices) == 0 {
		c.BootDevices = []string{"hd"}
	} else {
		for _, bd := range c.BootDevices {
			switch bd {
			case "hd":
				continue
			case "network":
				continue
			case "cdrom":
				continue
			default:
				errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("unknown BootDevice: %s", bd))
			}
		}
	}

	for i, volumeDef := range c.Volumes {
		w, e := volumeDef.PrepareConfig(&c.ctx, c.DomainName)
		warnings = append(warnings, w...)
		errs = packersdk.MultiErrorAppend(errs, e...)
		c.Volumes[i] = volumeDef
	}

	for i, ni := range c.NetworkInterfaces {
		w, e := ni.PrepareConfig(&c.ctx)
		warnings = append(warnings, w...)
		errs = packersdk.MultiErrorAppend(errs, e...)
		c.NetworkInterfaces[i] = ni
	}

	warnings, errs = c.prepareCommunicator(warnings, errs)

	if len(c.Volumes) == 0 {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("no volume has been specified"))
	} else {
		errs, warnings = c.prepareArtifactVolume(errs, warnings)
	}

	if c.NetworkAddressSource == "" {
		c.NetworkAddressSource = "agent"
		warnings = append(warnings, "No network_address_source was specified, defaulting to agent. This might hang your build when there are no qemu agent running on the builder machine")
	}

	if _, err := mapNetworkAddressSources(c.NetworkAddressSource); err != nil {
		errs = packersdk.MultiErrorAppend(errs, err)
	}

	if c.ShutdownMode == "" {
		c.ShutdownMode = "auto"
	}

	if _, ok := mapDomainShutdown(c.ShutdownMode); !ok {
		err := fmt.Errorf("unrecognized shutdown mode '%s'", c.ShutdownMode)
		errs = packersdk.MultiErrorAppend(errs, err)
	}

	if c.ShutdownTimeout <= 0 {
		c.ShutdownTimeout = 5 * time.Minute
	}

	if len(errs.Errors) > 0 {
		return warnings, errs
	}

	if c.DomainType == "" {
		c.DomainType = "kvm"
	}

	if c.Arch == "" {
		c.Arch = "x86_64"
	}

	return warnings, nil
}

func (c *Config) prepareArtifactVolume(errs *packersdk.MultiError, warnings []string) (*packersdk.MultiError, []string) {
	original := c.ArtifactVolumeAlias
	if original == "" {
		c.ArtifactVolumeAlias = "artifact"
	}

	c.ArtifactVolumeAlias = fmt.Sprintf("ua-%s", c.ArtifactVolumeAlias)

	found := false
	for _, vol := range c.Volumes {

		if vol.Alias == c.ArtifactVolumeAlias {
			found = true
			break
		}
	}
	if !found {
		if original != "" {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("no volume found with alias '%s'", original))
		} else if len(c.Volumes) == 1 {
			warnings = append(warnings, "Using the only defined volume as an artifact")
			vol0 := c.Volumes[0]
			vol0.Alias = c.ArtifactVolumeAlias
			c.Volumes[0] = vol0
		} else {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("please specify an alias for a volume and set artifact_volume_alias on the builder"))
		}
	}
	return errs, warnings
}

func (c *Config) prepareCommunicator(warnings []string, errs *packersdk.MultiError) ([]string, *packersdk.MultiError) {
	original := c.CommunicatorInterface
	if c.CommunicatorInterface == "" {
		c.CommunicatorInterface = "communicator"
	}

	// Libvirt only accepts alias with the prefix `ua-`
	c.CommunicatorInterface = fmt.Sprintf("ua-%s", c.CommunicatorInterface)

	if len(c.NetworkInterfaces) == 0 {
		warnings = append(warnings, "No network interface defined")
	} else {
		found := false
		for _, ni := range c.NetworkInterfaces {
			if ni.Alias == c.CommunicatorInterface {
				found = true
				break
			}
		}
		if !found {
			if original != "" {
				errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("no network_interface found with alias '%s'", original))
			} else {
				warnings = append(warnings, "using first network interface found as communicator interface")
				ni0 := c.NetworkInterfaces[0]
				ni0.Alias = c.CommunicatorInterface
				c.NetworkInterfaces[0] = ni0
			}
		}
	}
	return warnings, errs
}
