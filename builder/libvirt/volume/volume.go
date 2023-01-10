//go:generate go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc struct-markdown

package volume

import (
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/rs/xid"
	"libvirt.org/go/libvirtxml"
)

type Volume struct {
	// Specifies the name of the storage pool (managed by libvirt) where the disk resides. If not specified
	// the pool named `default` will be used
	Pool string `mapstructure:"pool" required:"false"`
	// Providing a name for the volume which is unique to the pool. (For a disk pool, the name must be
	// combination of the source device path device and next partition number to be
	// created. For example, if the source device path is /dev/sdb and there are no
	// partitions on the disk, then the name must be sdb1 with the next name being
	// sdb2 and so on.)
	// If not provided, a name will be generated in a format of `<domain_name>-<postfix>` where
	// postfix will be either the alias or a random id.
	// For cloudinit, the generated name will be `<domain_name>-cloudinit`
	Name string `mapstructure:"name" required:"false"`
	// Provides information about the underlying storage allocation of the volume.
	// This may not be available for some pool types.
	Source *VolumeSource `mapstructure:"source" required:"false"`
	// Providing the total storage allocation for the volume.
	// This may be smaller than the logical capacity if the volume is sparsely allocated.
	// It may also be larger than the logical capacity if the volume has substantial metadata overhead.
	// If omitted when creating a volume, the volume will be fully allocated at time of creation.
	// If set to a value smaller than the capacity, the pool has the option of deciding to sparsely allocate a volume.
	// It does not have to honour requests for sparse allocation though.
	// Different types of pools may treat sparse volumes differently.
	// For example, the logical pool will not automatically expand volume's allocation when it gets full;
	Size string `mapstructure:"size" required:"false"`
	// Providing the logical capacity for the volume. This value is in bytes by default,
	// but a unit attribute can be specified with the same semantics as for allocation
	// This is compulsory when creating a volume.
	Capacity string `mapstructure:"capacity" required:"false"`
	// If true, it indicates the device cannot be modified by the guest.
	ReadOnly bool `mapstructure:"readonly" required:"false"`
	// The target element controls the bus / device under which the disk is exposed
	// to the guest OS. The dev attribute indicates the "logical" device name.
	// The actual device name specified is not guaranteed to map to the device name
	// in the guest OS. Treat it as a device ordering hint.
	// If no `bus` and `target_dev` attribute is specified, it defaults to `target_dev = "sda"` with `bus = "scsi"`
	TargetDev string `mapstructure:"target_dev" required:"false"`
	// The optional bus attribute specifies the type of disk device to emulate;
	// possible values are driver specific, with typical values being
	// `ide`, `scsi`, `virtio`, `xen`, `usb`, `sata`, or `sd` `sd` since 1.1.2.
	// If omitted, the bus type is inferred from the style of the device name
	// (e.g. a device named 'sda' will typically be exported using a SCSI bus).
	Bus string `mapstructure:"bus" required:"false"`
	// To help users identifying devices they care about, every device can have an alias which must be unique within the domain.
	// Additionally, the identifier must consist only of the following characters: `[a-zA-Z0-9_-]`.
	Alias string `mapstructure:"alias" required:"false"`
	// Specifies the volume format type, like `qcow`, `qcow2`, `vmdk`, `raw`. If omitted, the storage pool's default format
	// will be used.
	Format string `mapstructure:"format" required:"false"`

	allowUnspecifiedSize bool `undocumented:"true"`
}

func (v *Volume) PrepareConfig(ctx *interpolate.Context, domainName string) (warnings []string, errs []error) {
	warnings = []string{}
	errs = []error{}

	log.Println("Preparing volume config")

	if v.Alias != "" {
		v.Alias = fmt.Sprintf("ua-%s", v.Alias)
	}

	if v.Pool == "" {
		v.Pool = "default"
		warnings = append(warnings, fmt.Sprintf("Pool isn't set for volume %s, using the '%s' pool", v.Name, v.Pool))
	}

	if v.Bus == "" && v.TargetDev == "" {
		v.Bus = "scsi"
		v.TargetDev = "sda"
		warnings = append(warnings, fmt.Sprintf("Bus and target_dev aren't set for volume %s/%s, using bus=%s and target_dev=%s as default", v.Pool, v.Name, v.Bus, v.TargetDev))
	}

	if v.Source != nil {
		w, e := v.Source.PrepareConfig(ctx, v, domainName)
		warnings = append(warnings, w...)
		errs = append(errs, e...)
	}

	if v.Name == "" {
		postfix := v.Alias

		if postfix == "" {
			postfix = xid.New().String()
		}
		v.Name = fmt.Sprintf("%s-%s", domainName, postfix)
		warnings = append(warnings, fmt.Sprintf("Volume name was not set, using '%s' as volume name instead.", v.Name))
	}

	if v.Size == "" && v.Capacity == "" && !v.allowUnspecifiedSize {
		errs = append(errs, fmt.Errorf("at least one of Volume.Size or Volume.Capacity must be set for volume %s/%s", v.Pool, v.Name))
	}

	if v.Size == "" {
		v.Size = v.Capacity
	}
	if v.Capacity == "" {
		v.Capacity = v.Size
	}

	return
}

func (v *Volume) StorageDefinitionXml() (*libvirtxml.StorageVolume, error) {
	storageDef := &libvirtxml.StorageVolume{
		Name: v.Name,
	}

	if v.Capacity != "" {
		capVal, capUnit, err := fmtReadPostfixedValue(v.Capacity)

		if err != nil {
			return nil, fmt.Errorf("couldn't understand volume capacity '%s' : %s", v.Capacity, err)
		}
		storageDef.Capacity = &libvirtxml.StorageVolumeSize{
			Value: capVal,
			Unit:  capUnit,
		}
	}

	if v.Size != "" {
		allocVal, allocUnit, err := fmtReadPostfixedValue(v.Size)

		if err != nil {
			return nil, fmt.Errorf("couldn't understand volume size '%s' : %s", v.Size, err)
		}

		storageDef.Allocation = &libvirtxml.StorageVolumeSize{
			Value: allocVal,
			Unit:  allocUnit,
		}
	}

	if v.Format != "" {
		storageDef.Target = &libvirtxml.StorageVolumeTarget{
			Format: &libvirtxml.StorageVolumeTargetFormat{Type: v.Format},
		}
	}

	if v.Source != nil {
		v.Source.UpdateStorageDefinitionXml(storageDef)
	}
	return storageDef, nil
}

func (v *Volume) DomainDiskXml() *libvirtxml.DomainDisk {
	domainDisk := &libvirtxml.DomainDisk{
		Device: "disk",
		Source: &libvirtxml.DomainDiskSource{
			Volume: &libvirtxml.DomainDiskSourceVolume{
				Pool:   v.Pool,
				Volume: v.Name,
			},
		},
	}

	if v.Alias != "" {
		domainDisk.Alias = &libvirtxml.DomainAlias{
			Name: v.Alias,
		}
	}

	if v.Bus != "" {
		if domainDisk.Target == nil {
			domainDisk.Target = &libvirtxml.DomainDiskTarget{}
		}
		domainDisk.Target.Bus = v.Bus
	}

	if v.TargetDev != "" {
		if domainDisk.Target == nil {
			domainDisk.Target = &libvirtxml.DomainDiskTarget{}
		}
		domainDisk.Target.Dev = v.TargetDev
	}

	if v.Format != "" {
		domainDisk.Driver = &libvirtxml.DomainDiskDriver{}
		domainDisk.Driver.Type = v.Format
	}

	if v.ReadOnly {
		domainDisk.ReadOnly = &libvirtxml.DomainDiskReadOnly{}
	}

	if v.Source != nil {
		v.Source.UpdateDomainDiskXml(domainDisk)
	}
	return domainDisk
}

func (v *Volume) PrepareVolume(pctx *PreparationContext) multistep.StepAction {
	pctx.Ui.Message(fmt.Sprintf("Preparing volume %s/%s", v.Pool, v.Name))

	pctx.VolumeConfig = v

	pool, err := pctx.Driver.StoragePoolLookupByName(pctx.VolumeConfig.Pool)
	if err != nil {
		return pctx.HaltOnError(err, "Error while looking up storage pool %s: %s", pctx.VolumeConfig.Pool, err)
	}
	pctx.PoolRef = &pool

	volumeDef, err := v.StorageDefinitionXml()
	if err != nil {
		return pctx.HaltOnError(err, "Couldn't produce volume definition XML for %s/%s: %s", v.Pool, v.Name, err)
	}

	pctx.VolumeDefinition = volumeDef

	if pctx.VolumeConfig.Source == nil {
		vol, err := pctx.Driver.StorageVolLookupByName(*pctx.PoolRef, pctx.VolumeConfig.Name)

		if err == nil {
			pctx.VolumeRef = &vol
			pctx.VolumeIsCreated = false
		} else if !pctx.VolumeIsArtifact {
			return pctx.HaltOnError(err, "Error while looking up volume %s/%s: %s", v.Pool, v.Name, err)
		}
	} else {
		action := pctx.VolumeConfig.Source.PrepareVolume(pctx)
		if action != multistep.ActionContinue {
			return action
		}
	}

	if pctx.VolumeRef == nil {
		err := pctx.CreateVolume()
		if err != nil {
			return pctx.HaltOnError(err, "Error while creating volume %s/%s: %s", v.Pool, v.Name, err)
		}
	}

	return multistep.ActionContinue
}

func fmtReadPostfixedValue(s string) (value uint64, unit string, err error) {
	var n int
	n, err = fmt.Sscanf(s, "%d%s", &value, &unit)
	if err != nil {
		return 0, "", err
	}
	if n != 2 {
		unit = "B"
	}
	switch unit {
	case "":
		unit = "B"
	case "k", "kb", "kB":
		unit = "KiB"
	case "M", "Mb", "MB":
		unit = "MiB"
	case "G", "Gb", "GB":
		unit = "Gib"
	}
	return
}
