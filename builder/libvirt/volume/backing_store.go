//go:generate packer-sdc struct-markdown

package volume

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

type BackingStoreVolumeSource struct {
	// Specifies the name of the storage pool (managed by libvirt) where the disk source resides
	Pool string `mapstucture:"pool" required:"false"`
	// Specifies the name of storage volume (managed by libvirt) used as the disk source.
	Volume string `mapstructure:"volume" required:"false"`
	// The file backing the volume. Mutually exclusive with pool and volume args!
	Path string `mapstructure:"path" required:"false"`
}

func (vs *BackingStoreVolumeSource) PrepareConfig(ctx *interpolate.Context, vol *Volume) (warnings []string, errs []error) {
	errs = []error{}

	if vs.Path == "" {
		if vs.Pool == "" {
			errs = append(errs, fmt.Errorf("backing store volume missing pool name"))
		}
		if vs.Volume == "" {
			errs = append(errs, fmt.Errorf("backing store volume missing volume name"))
		}
	} else {
		if vs.Pool != "" || vs.Volume != "" {
			errs = append(errs, fmt.Errorf("if path is specified on a backing store, no other arguments can be specified"))
		}
	}

	return
}

func (vs *BackingStoreVolumeSource) UpdateDomainDiskXml(domainDisk *libvirtxml.DomainDisk) {
	if domainDisk.Driver == nil {
		domainDisk.Driver = &libvirtxml.DomainDiskDriver{}
	}
	domainDisk.Driver.Type = "qcow2"
}

func (vs *BackingStoreVolumeSource) UpdateStorageDefinitionXml(storageDef *libvirtxml.StorageVolume) {
	if storageDef.Target == nil {
		storageDef.Target = &libvirtxml.StorageVolumeTarget{}
	}
	if storageDef.Target.Format == nil {
		storageDef.Target.Format = &libvirtxml.StorageVolumeTargetFormat{}
	}
	storageDef.Target.Format.Type = "qcow2"
}

func (vs *BackingStoreVolumeSource) PrepareVolume(pctx *PreparationContext) multistep.StepAction {
	backing_pool, err := pctx.Driver.StoragePoolLookupByName(vs.Pool)
	if err != nil {
		return pctx.HaltOnError(err, "BackingStoreSource.PoolLookup: %s", err)
	}

	backing_vol, err := pctx.Driver.StorageVolLookupByName(backing_pool, vs.Volume)
	if err != nil {
		return pctx.HaltOnError(err, "BackingStoreSource.VolumeLookup: %s", err)
	}

	raw_xml, err := pctx.Driver.StorageVolGetXMLDesc(backing_vol, 0)
	if err != nil {
		return pctx.HaltOnError(err, "BackingStoreSource.GetXMLDescription: %s", err)
	}

	backing_vol_def := &libvirtxml.StorageVolume{}
	err = backing_vol_def.Unmarshal(raw_xml)

	if err != nil {
		return pctx.HaltOnError(err, "BackingStoreSource.Unmarshal: %s", err)
	}

	volumeDef := pctx.VolumeDefinition
	volumeDef.BackingStore = &libvirtxml.StorageVolumeBackingStore{
		Path:        backing_vol_def.Target.Path,
		Format:      backing_vol_def.Target.Format,
		Permissions: backing_vol_def.Target.Permissions,
	}

	return multistep.ActionContinue
}
