//go:generate go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc struct-markdown

package volume

import (
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type CloningVolumeSource struct {
	// Specifies the name of the storage pool (managed by libvirt) where the disk source resides
	Pool string `mapstucture:"pool" required:"false"`
	// Specifies the name of storage volume (managed by libvirt) used as the disk source.
	Volume string `mapstructure:"volume" required:"false"`
}

func (vs *CloningVolumeSource) PrepareConfig(ctx *interpolate.Context, vol *Volume) (warnings []string, errs []error) {
	errs = []error{}

	if vs.Pool == "" {
		errs = append(errs, fmt.Errorf("cloning volume missing pool name"))
	}
	if vs.Volume == "" {
		errs = append(errs, fmt.Errorf("cloning store volume missing volume name"))
	}

	vol.allowUnspecifiedSize = true

	return
}

func (vs *CloningVolumeSource) UpdateDomainDiskXml(domainDisk *libvirtxml.DomainDisk) {
	if domainDisk.Driver == nil {
		domainDisk.Driver = &libvirtxml.DomainDiskDriver{}
	}
	domainDisk.Driver.Type = "qcow2"
}

func (vs *CloningVolumeSource) UpdateStorageDefinitionXml(storageDef *libvirtxml.StorageVolume) {
	if storageDef.Target == nil {
		storageDef.Target = &libvirtxml.StorageVolumeTarget{}
	}
	if storageDef.Target.Format == nil {
		storageDef.Target.Format = &libvirtxml.StorageVolumeTargetFormat{}
	}
	storageDef.Target.Format.Type = "qcow2"
}

func (vs *CloningVolumeSource) PrepareVolume(pctx *PreparationContext) multistep.StepAction {
	backingPool, err := pctx.Driver.StoragePoolLookupByName(vs.Pool)
	if err != nil {
		return pctx.HaltOnError(err, "CloningVolumeSource.PoolLookup: %s", err)
	}

	backingVol, err := pctx.Driver.StorageVolLookupByName(backingPool, vs.Volume)
	if err != nil {
		return pctx.HaltOnError(err, "CloningVolumeSource.VolumeLookup: %s", err)
	}

	rawXML, err := pctx.Driver.StorageVolGetXMLDesc(backingVol, 0)
	if err != nil {
		return pctx.HaltOnError(err, "CloningVolumeSource.GetXMLDescription: %s", err)
	}

	backingVolDef := &libvirtxml.StorageVolume{}
	err = backingVolDef.Unmarshal(rawXML)

	if err != nil {
		return pctx.HaltOnError(err, "CloningVolumeSource.Unmarshal: %s", err)
	}

	err = pctx.CloneVolumeFrom(backingPool, backingVol)

	if err != nil {
		return pctx.HaltOnError(err, "%s", err)
	}

	err = pctx.RefreshVolumeDefinition()

	if err != nil {
		log.Printf("Error while refreshing volume definition: %s\n", err)
	}

	return multistep.ActionContinue
}
