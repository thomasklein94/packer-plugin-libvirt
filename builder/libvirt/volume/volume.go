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

var nextSerialLetter func() (string, error)
var nextIdeLetter func() (string, error)
var nextFloppyLetter func() (string, error)
var nextVirtioLetter func() (string, error)

func letterGenerator(prefix string, options string) func() (string, error) {
	if len(options) == 0 {
		panic("letterGenerator needs at least 1 option to work with")
	}

	var next = 0
	return func() (string, error) {
		if next >= len(options) {
			return "", fmt.Errorf("device bus has used up all possible letters")
		}
		nextLetter := options[next]
		next = next + 1

		return prefix + string(nextLetter), nil
	}
}

const asciiLower = "abcdefghijklmnopqrstuvwxyz"

const scsiSataPrefix = "sd"
const idePrefix = "hd"
const floppyPrefix = "fd"
const virtualPrefix = "vd"

func ResetDeviceLetters() {
	nextSerialLetter = letterGenerator(scsiSataPrefix, asciiLower)
	nextIdeLetter = letterGenerator(idePrefix, asciiLower)
	nextFloppyLetter = letterGenerator(floppyPrefix, "abcd")
	nextVirtioLetter = letterGenerator(virtualPrefix, asciiLower)
}

func init() {
	ResetDeviceLetters()
}

func nextDriveLetter(bus string) (string, error) {
	switch bus {
	case "fdc":
		return nextFloppyLetter()
	case "ide":
		return nextIdeLetter()
	case "virtio":
		return nextVirtioLetter()
	default:
		return nextSerialLetter()
	}
}

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
	// DEPRECATED!
	// The device order on the bus is determined automatically from the order of definition
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
	// Specifies the device type. If omitted, defaults to "disk". Can be `disk`, `floppy`, `cdrom` or `lun`.
	Device string `mapstructure:"device" required:"false"`

	allowUnspecifiedSize bool `undocumented:"true"`
}

func (v *Volume) PrepareConfig(ctx *interpolate.Context, domainName string) (warnings []string, errs []error) {
	warnings = []string{}
	errs = []error{}

	log.Println("Preparing volume config")

	if v.Device == "" {
		v.Device = "disk"
	}

	if v.Alias != "" {
		v.Alias = fmt.Sprintf("ua-%s", v.Alias)
	}

	if v.Pool == "" {
		v.Pool = "default"
		warnings = append(warnings, fmt.Sprintf("Pool isn't set for volume, using the '%s' pool", v.Pool))
	}

	if v.Bus == "" {
		if v.Device == "floppy" {
			v.Bus = "fdc"
		} else {
			v.Bus = "scsi"
		}
		warnings = append(warnings, fmt.Sprintf("Bus isn't set, using bus=%s as default", v.Bus))
	}

	targetDev, err := nextDriveLetter(v.Bus)
	if v.TargetDev != "" {
		warnings = append(warnings, fmt.Sprintf(`
			Setting target_dev for volume is deprecated.
			Order on the bus now determined by the order of the volume blocks!
			Please remove the target_dev attribute from the volume definition!
			Overriding target_dev='%s' to '%s'`, v.TargetDev, targetDev))
	}

	v.TargetDev = targetDev
	if err != nil {
		errs = append(errs, err)
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
		Device: v.Device,
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
		unit = "GiB"
	}
	return
}

func unitToMultiplier(unit string) (value int64, err error) {
	switch unit {
	case "B":
		value = 1
	case "KiB":
		value = 1024
	case "MiB":
		value = 1024 * 1024
	case "GiB":
		value = 1024 * 1024 * 1024
	default:
		err = fmt.Errorf("unknown unit %s", unit)
	}
	return
}
