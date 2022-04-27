//go:generate packer-sdc struct-markdown

package volume

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	libvirtxml "github.com/libvirt/libvirt-go-xml"

	"github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type HttpVolumeSource struct {
	// [required] An URL for installation image
	Url string `mapstructure:"url" required:"false"`
	// [required] The storage format type, eg `cow`, `qcow`, `qcow2`, `vmdk`, `raw`.
	Format string `mapstructure:"format" required:"false"`
}

func (vs *HttpVolumeSource) PrepareConfig(ctx *interpolate.Context, vol *Volume) (warnings []string, errs []error) {
	if vs.Format == "" {
		vs.Format = "qcow2"
		warnings = append(warnings, fmt.Sprintf("Http source format didn't set, using %s", vs.Format))
	}
	return
}

func (vs *HttpVolumeSource) UpdateDomainDiskXml(domainDisk *libvirtxml.DomainDisk) {
	if domainDisk.Driver == nil {
		domainDisk.Driver = &libvirtxml.DomainDiskDriver{}
	}
	domainDisk.Driver.Type = vs.Format
}

func (vs *HttpVolumeSource) UpdateStorageDefinitionXml(storageDef *libvirtxml.StorageVolume) {
	if storageDef.Target == nil {
		storageDef.Target = &libvirtxml.StorageVolumeTarget{}
	}
	if storageDef.Target.Format == nil {
		storageDef.Target.Format = &libvirtxml.StorageVolumeTargetFormat{}
	}
	storageDef.Target.Format.Type = vs.Format
}

func (vs *HttpVolumeSource) PrepareVolume(pctx *PreparationContext) multistep.StepAction {
	pctx.Ui.Message(fmt.Sprintf("Using volume source from URL: '%s'", vs.Url))

	response, err := http.Get(vs.Url)
	if err != nil {
		return pctx.HaltOnError(err, "Couldn't download image: %s", err)
	}
	var allocation uint64
	_, err = fmt.Sscan(response.Header.Get("Content-Length"), &allocation)
	if err != nil {
		return pctx.HaltOnError(err, "Can't parse '%s' as uint64 value for allocation: %s", response.Header.Get("Content-Length"), err)
	}

	pctx.VolumeDefinition.Allocation = &libvirtxml.StorageVolumeSize{
		Value: allocation,
		Unit:  "B",
	}

	err = pctx.CreateVolume()
	if err != nil {
		return pctx.HaltOnError(err, "%s", err)
	}

	err = pctx.Driver.StorageVolUpload(*pctx.VolumeRef, response.Body, 0, allocation, libvirt.StorageVolUploadSparseStream)

	if err != nil {
		connect_uri, _ := pctx.Driver.ConnectGetUri()

		// The test backend does not support Volume Uploads, so
		if connect_uri[0:4] == "test" {
			pctx.Ui.Error(fmt.Sprintf("Error during volume streaming: %s", err))
		} else {
			return pctx.HaltOnError(err, "Error during volume streaming: %s", err)
		}
	}

	err = pctx.RefreshVolumeDefinition()

	if err != nil {
		log.Printf("Error while refreshing volume definition: %s\n", err)
	}

	return multistep.ActionContinue
}
