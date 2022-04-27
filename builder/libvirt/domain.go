package libvirt

import (
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

func newDomainDefinition(config *Config) libvirtxml.Domain {
	domainDef := libvirtxml.Domain{
		Name:        config.DomainName,
		Description: "Domain created by packer-plugin-libvirt",
		Type:        "kvm",
		Memory: &libvirtxml.DomainMemory{
			Value: uint(config.MemorySize),
			Unit:  "MiB",
		},
		VCPU: &libvirtxml.DomainVCPU{
			Placement: "static",
			Value:     uint(config.CpuCount),
		},
		Devices: &libvirtxml.DomainDeviceList{
			Channels: []libvirtxml.DomainChannel{
				{
					Source: &libvirtxml.DomainChardevSource{
						UNIX: &libvirtxml.DomainChardevSourceUNIX{},
					},
					Target: &libvirtxml.DomainChannelTarget{
						VirtIO: &libvirtxml.DomainChannelTargetVirtIO{
							Name: "org.qemu.guest_agent.0",
						},
					},
				},
			},
			Serials: []libvirtxml.DomainSerial{
				{
					Alias: &libvirtxml.DomainAlias{
						Name: "ua-serial-console",
					},
				},
			},
			Consoles: []libvirtxml.DomainConsole{
				{
					Alias: &libvirtxml.DomainAlias{
						Name: "ua-virtual-console",
					},
					Target: &libvirtxml.DomainConsoleTarget{
						Type: "virtio",
					},
				},
			},
			Disks:      []libvirtxml.DomainDisk{},
			Graphics:   []libvirtxml.DomainGraphic{},
			Interfaces: []libvirtxml.DomainInterface{},
		},
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Arch: "x86_64",
				Type: "hvm",
			},
		},
		Features: &libvirtxml.DomainFeatureList{
			PAE:  &libvirtxml.DomainFeature{},
			ACPI: &libvirtxml.DomainFeature{},
			APIC: &libvirtxml.DomainFeatureAPIC{},
		},
	}

	for _, dg := range config.DomainGraphics {
		domainDef.Devices.Graphics = append(domainDef.Devices.Graphics, *dg.DomainGraphic())
	}

	if len(domainDef.Devices.Graphics) > 0 {
		domainDef.Devices.Videos = []libvirtxml.DomainVideo{
			{
				Model: libvirtxml.DomainVideoModel{
					Type: "virtio",
				},
			},
		}
	}

	for _, ni := range config.NetworkInterfaces {
		domainDef.Devices.Interfaces = append(domainDef.Devices.Interfaces, *ni.DomainInterface())
	}

	return domainDef
}
