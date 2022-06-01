//go:generate packer-sdc struct-markdown

package volume

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

type VolumeSource struct {
	Type          string                   `mapstructure:"type" required:"true"`
	External      ExternalVolumeSource     `mapstructure:",squash"`
	CloudInit     CloudInitSource          `mapstructure:",squash"`
	BackingStore  BackingStoreVolumeSource `mapstructure:",squash"`
	CloningVolume CloningVolumeSource      `mapstructure:",squash"`
}

func (vs *VolumeSource) PrepareConfig(ctx *interpolate.Context, vol *Volume, domainName string) (warnings []string, errs []error) {
	switch vs.Type {
	case "external":
		return vs.External.PrepareConfig(ctx, vol)
	case "cloud-init", "cloudinit":
		return vs.CloudInit.PrepareConfig(ctx, vol, domainName)
	case "backing-store", "backingstore":
		return vs.BackingStore.PrepareConfig(ctx, vol)
	case "cloning", "clone":
		vs.CloningVolume.PrepareConfig(ctx, vol)
	default:
		errs = append(errs, fmt.Errorf("unsupported volume source type '%s'", vs.Type))
	}
	return
}

func (vs *VolumeSource) UpdateDomainDiskXml(domainDisk *libvirtxml.DomainDisk) {
	switch vs.Type {
	case "external":
		vs.External.UpdateDomainDiskXml(domainDisk)
	case "cloud-init", "cloudinit":
		vs.CloudInit.UpdateDomainDiskXml(domainDisk)
	case "backing-store", "backingstore":
		vs.BackingStore.UpdateDomainDiskXml(domainDisk)
	case "cloning", "clone":
		vs.CloningVolume.UpdateDomainDiskXml(domainDisk)
	}
}

func (vs *VolumeSource) UpdateStorageDefinitionXml(storageDef *libvirtxml.StorageVolume) {
	switch vs.Type {
	case "external":
		vs.External.UpdateStorageDefinitionXml(storageDef)
	case "cloud-init", "cloudinit":
		vs.CloudInit.UpdateStorageDefinitionXml(storageDef)
	case "backing-store", "backingstore":
		vs.BackingStore.UpdateStorageDefinitionXml(storageDef)
	case "cloning", "clone":
		vs.CloningVolume.UpdateStorageDefinitionXml(storageDef)
	}
}

func (vs *VolumeSource) PrepareVolume(pctx *PreparationContext) multistep.StepAction {
	switch vs.Type {
	case "external":
		return vs.External.PrepareVolume(pctx)
	case "cloud-init", "cloudinit":
		return vs.CloudInit.PrepareVolume(pctx)
	case "backing-store", "backingstore":
		return vs.BackingStore.PrepareVolume(pctx)
	case "cloning", "clone":
		vs.CloningVolume.PrepareVolume(pctx)
	}
	return multistep.ActionContinue
}
