//go:generate go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc struct-markdown

package volume

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type CloudInitSource struct {
	// [optional] NoCloud instance metadata as string.
	// If not present, a default one will be generated with the instance-id set to the same value as the domain name
	// If you want to set network config, see the `network_config` attribute
	MetaData *string `mapstructure:"meta_data"`
	// [optional] CloudInit user-data as string.
	// See [cloud-init documentation](https://cloudinit.readthedocs.io/en/latest/topics/format.html) for more
	UserData *string `mapstructure:"user_data"`
	// [optional] Network configuration for cloud-init to be picked up.
	// User-data cannot change an instance’s network configuration.
	// In the absence of network configuration in any sources,
	// Cloud-init will write out a network configuration that will issue a DHCP request on a “first” network interface.
	// Read more about the possible format and features at
	// [cloud-init documentation](https://cloudinit.readthedocs.io/en/latest/topics/network-config.html).
	NetworkConfig *string `mapstructure:"network_config"`
}

func (vs *CloudInitSource) PrepareConfig(ctx *interpolate.Context, vol *Volume, domainName string) (warnings []string, errs []error) {
	if vol.Name == "" {
		vol.Name = fmt.Sprintf("%s-cloudinit", domainName)
	}

	vol.allowUnspecifiedSize = true

	return
}

func (vs *CloudInitSource) UpdateDomainDiskXml(domainDisk *libvirtxml.DomainDisk) {
	if domainDisk.Driver == nil {
		domainDisk.Driver = &libvirtxml.DomainDiskDriver{}
	}
	if domainDisk.ReadOnly == nil {
		domainDisk.ReadOnly = &libvirtxml.DomainDiskReadOnly{}
	}

	domainDisk.Driver.Type = "raw"
	domainDisk.Device = "cdrom"
}

func (vs *CloudInitSource) UpdateStorageDefinitionXml(storageDef *libvirtxml.StorageVolume) {
	if storageDef.Target == nil {
		storageDef.Target = &libvirtxml.StorageVolumeTarget{}
	}
	if storageDef.Target.Format == nil {
		storageDef.Target.Format = &libvirtxml.StorageVolumeTargetFormat{}
	}
	storageDef.Target.Format.Type = "iso"
}

func (vs *CloudInitSource) PrepareVolume(pctx *PreparationContext) multistep.StepAction {
	// We are reusing CreateCD step in a bit hacky way to avoid coding this (for now)
	// A better solution to this should be implemented later

	pctx.Ui.Message(fmt.Sprintf("Assembling CloudInit image %s/%s", pctx.VolumeConfig.Pool, pctx.VolumeConfig.Name))
	createStep := commonsteps.StepCreateCD{
		Label:   "cidata",
		Content: map[string]string{},
	}

	if vs.MetaData == nil {
		domainDef := pctx.State.Get("domain_def").(*libvirtxml.Domain)
		defaultMetadata := fmt.Sprintf("instance_id: %s\n", domainDef.Name)

		vs.MetaData = &defaultMetadata
	}

	if *vs.MetaData != "" {
		createStep.Content["meta-data"] = *vs.MetaData
	}
	if vs.UserData != nil {
		createStep.Content["user-data"] = *vs.UserData
	}
	if vs.NetworkConfig != nil {
		createStep.Content["network-config"] = *vs.NetworkConfig
	}

	tempState := &multistep.BasicStateBag{}
	tempState.Put("ui", pctx.Ui)

	action := createStep.Run(pctx.Context, tempState)
	if action != multistep.ActionContinue {
		err := fmt.Errorf("unknown Error")
		rawStepError, ok := tempState.GetOk("error")
		if ok {
			err = rawStepError.(error)
		}
		return pctx.HaltOnError(err, "Couldn't produce a CloudInit ISO: %s", err)
	}
	cdPath := tempState.Get("cd_path").(string)

	defer createStep.Cleanup(tempState)

	return pctx.uploadVolume(cdPath)
}
