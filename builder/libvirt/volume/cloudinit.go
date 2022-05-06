//go:generate packer-sdc struct-markdown

package volume

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
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

func (vs *CloudInitSource) PrepareConfig(ctx *interpolate.Context, vol *Volume, domain_name string) (warnings []string, errs []error) {
	if vol.Name == "" {
		vol.Name = fmt.Sprintf("%s-cloudinit", domain_name)
	}

	vol.allow_unspecified_size = true

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
	create_step := commonsteps.StepCreateCD{
		Label:   "cidata",
		Content: map[string]string{},
	}

	if vs.MetaData == nil {
		domainDef := pctx.State.Get("domain_def").(*libvirtxml.Domain)
		default_metadata := fmt.Sprintf("instance_id: %s\n", domainDef.Name)

		vs.MetaData = &default_metadata
	}

	if *vs.MetaData != "" {
		create_step.Content["meta-data"] = *vs.MetaData
	}
	if vs.UserData != nil {
		create_step.Content["user-data"] = *vs.UserData
	}
	if vs.NetworkConfig != nil {
		create_step.Content["network-config"] = *vs.NetworkConfig
	}

	// temp_state := new(multistep.StateBag)
	temp_state := &multistep.BasicStateBag{}
	temp_state.Put("ui", pctx.Ui)

	action := create_step.Run(pctx.Context, temp_state)
	if action != multistep.ActionContinue {
		err := fmt.Errorf("unknown Error")
		raw_step_error, ok := temp_state.GetOk("error")
		if ok {
			err = raw_step_error.(error)
		}
		return pctx.HaltOnError(err, "Couldn't produce a CloudInit ISO: %s", err)
	}
	cd_path := temp_state.Get("cd_path").(string)

	fptr, err := os.Open(cd_path)
	if err != nil {
		pctx.HaltOnError(err, "CloudInit.Open: %s", err)
	}

	finfo, err := fptr.Stat()
	if err != nil {
		pctx.HaltOnError(err, "CloudInit.Stat: %s", err)
	}

	size := uint64(finfo.Size())
	pctx.VolumeDefinition.Capacity = &libvirtxml.StorageVolumeSize{
		Value: size,
		Unit:  "B",
	}
	// If omitted when creating a volume, the volume will be fully allocated at time of creation.
	pctx.VolumeDefinition.Allocation = nil

	err = pctx.CreateVolume()
	if err != nil {
		return pctx.HaltOnError(err, "%s", err)
	}

	err = pctx.Driver.StorageVolUpload(*pctx.VolumeRef, fptr, 0, size, 0)

	if err != nil {
		connect_uri, _ := pctx.Driver.ConnectGetUri()

		// The test backend does not support Volume Uploads, so
		if connect_uri[0:4] == "test" {
			pctx.Ui.Error(fmt.Sprintf("CloudInit.Upload: %s", err))
		} else {
			return pctx.HaltOnError(err, "CloudInit.Upload: %s", err)
		}
	}

	err = pctx.RefreshVolumeDefinition()

	if err != nil {
		log.Printf("Error while refreshing volume definition: %s\n", err)
	}

	if err := fptr.Close(); err != nil {
		log.Println("Error closing CD stream")
	}

	create_step.Cleanup(temp_state)

	return multistep.ActionContinue
}
