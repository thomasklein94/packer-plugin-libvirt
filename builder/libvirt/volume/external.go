//go:generate go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc struct-markdown

package volume

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"

	"github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

// An external volume source defines a volume source that is external (or remote) to
// the libvirt hypervisor. This usually means a cloud image or installation media
// from the web or local machine. It's using go-getter under the hood with support
// for HTTP, HTTPS, S3, SMB and local files. The files will be cached by Packer.
type ExternalVolumeSource struct {
	// The checksum and the type of the checksum for the download
	Checksum string `mapstructure:"checksum"`
	// A list of URLs from where this volume can be obtained
	Urls []string `mapstructure:"urls"`
}

func (vs *ExternalVolumeSource) PrepareConfig(ctx *interpolate.Context, vol *Volume) (warnings []string, errs []error) {
	errs = []error{}
	if len(vs.Urls) == 0 {
		errs = append(errs, fmt.Errorf("at least 1 URL must be specified for an external volume source"))
	}

	vol.allowUnspecifiedSize = true

	return
}

func (vs *ExternalVolumeSource) UpdateDomainDiskXml(domainDisk *libvirtxml.DomainDisk) {}

func (vs *ExternalVolumeSource) UpdateStorageDefinitionXml(storageDef *libvirtxml.StorageVolume) {}

func (vs *ExternalVolumeSource) PrepareVolume(pctx *PreparationContext) multistep.StepAction {
	tmpState := multistep.BasicStateBag{}
	tmpState.Put("ui", pctx.Ui)
	resultKey := "path"

	step := commonsteps.StepDownload{
		Checksum:    vs.Checksum,
		Description: fmt.Sprintf("%s/%s", pctx.VolumeConfig.Pool, pctx.VolumeConfig.Name),
		Url:         vs.Urls,
		ResultKey:   resultKey,
	}

	action := step.Run(pctx.Context, &tmpState)

	if err, ok := tmpState.GetOk("error"); ok {
		return pctx.HaltOnError(err.(error), "%s", err)
	}

	if action != multistep.ActionContinue {
		return action
	}

	path := tmpState.Get(resultKey).(string)

	stat, err := os.Stat(path)
	if err != nil {
		return pctx.HaltOnError(err, "preparing volume: %s", err)
	}

	storageTargetCapacity := pctx.VolumeDefinition.Capacity

	pctx.VolumeDefinition.Allocation = &libvirtxml.StorageVolumeSize{
		Value: uint64(stat.Size()),
		Unit:  "B",
	}

	pctx.VolumeDefinition.Capacity = &libvirtxml.StorageVolumeSize{
		Value: uint64(stat.Size()),
		Unit:  "B",
	}

	err = pctx.CreateVolume()
	if err != nil {
		return pctx.HaltOnError(err, "%s", err)
	}

	f, err := os.Open(path)

	if err != nil {
		return pctx.HaltOnError(err, "preparing volume: %s", err)
	}

	err = pctx.Driver.StorageVolUpload(*pctx.VolumeRef, f, 0, uint64(stat.Size()), libvirt.StorageVolUploadSparseStream)

	if err != nil {
		connectUri, _ := pctx.Driver.ConnectGetUri()

		// The test backend does not support Volume Uploads, so
		if connectUri[0:4] == "test" {
			pctx.Ui.Error(fmt.Sprintf("Error during volume streaming: %s", err))
		} else {
			return pctx.HaltOnError(err, "Error during volume streaming: %s", err)
		}
	}

	err = pctx.RefreshVolumeDefinition()

	if err != nil {
		log.Printf("Error while refreshing volume definition: %s\n", err)
	}

	if storageTargetCapacity != nil {
		pctx.Ui.Message(fmt.Sprintf(
			"Resizing volume %s/%s to meet capacity %d%s",
			pctx.VolumeConfig.Pool,
			pctx.VolumeConfig.Name,
			storageTargetCapacity.Value,
			storageTargetCapacity.Unit,
		))
		multiplier, err := unitToMultiplier(storageTargetCapacity.Unit)
		if err != nil {
			return pctx.HaltOnError(err, "Error during volume resize: %s", err)
		}
		targetCapacityInBytes := storageTargetCapacity.Value * uint64(multiplier)

		err = pctx.Driver.StorageVolResize(*pctx.VolumeRef, targetCapacityInBytes, 0)
		if err != nil {
			return pctx.HaltOnError(err, "Error during volume resize: %s", err)
		}
	}

	return multistep.ActionContinue
}
