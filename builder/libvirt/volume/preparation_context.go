package volume

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"libvirt.org/go/libvirtxml"
)

type PreparationContext struct {
	State            multistep.StateBag
	Ui               packersdk.Ui
	Driver           *libvirt.Libvirt
	VolumeConfig     *Volume
	VolumeRef        *libvirt.StorageVol
	VolumeDefinition *libvirtxml.StorageVolume
	PoolRef          *libvirt.StoragePool
	VolumeIsCreated  bool
	VolumeIsArtifact bool
	Context          context.Context
}

func (pctx *PreparationContext) CreateVolume() error {
	if pctx.VolumeRef != nil {
		return fmt.Errorf("CreateVolume: Volume already exists")
	}

	volumeXML, err := pctx.VolumeDefinition.Marshal()
	if err != nil {
		return fmt.Errorf("CreateVolume.Marshal: %s", err)
	}

	if pctx.State.Get("debug").(bool) {
		log.Printf("Volume definition XML:\n%s\n", volumeXML)
	}

	ref, err := pctx.Driver.StorageVolCreateXML(*pctx.PoolRef, volumeXML, 0)
	if err != nil {
		return fmt.Errorf("CreateVolume.RPC: %s", err)
	}

	pctx.VolumeRef = &ref
	pctx.VolumeIsCreated = true

	return nil
}

func (pctx *PreparationContext) CloneVolumeFrom(sourcePool libvirt.StoragePool, sourceVol libvirt.StorageVol) error {
	if pctx.VolumeRef != nil {
		return fmt.Errorf("CreateVolumeFrom: Volume already exists")
	}

	if pctx.VolumeDefinition.BackingStore != nil {
		return fmt.Errorf("can't simultaneously clone a volume and use a backing store")
	}

	volumeXML, err := pctx.VolumeDefinition.Marshal()
	if err != nil {
		return fmt.Errorf("CreateVolumeFrom.Marshal: %s", err)
	}

	ref, err := pctx.Driver.StorageVolCreateXMLFrom(sourcePool, volumeXML, sourceVol, 0)
	if err != nil {
		return fmt.Errorf("CreateVolumeFrom.RPC: %s", err)
	}

	pctx.VolumeRef = &ref
	pctx.VolumeIsCreated = true

	return nil
}

func (pctx *PreparationContext) RefreshVolumeDefinition() error {
	rawVolumeDef, err := pctx.Driver.StorageVolGetXMLDesc(*pctx.VolumeRef, 0)
	if err != nil {
		return fmt.Errorf("RefreshVolumeDefinition.GetXMLDesc: %s", err)
	}

	volumeDef := &libvirtxml.StorageVolume{}
	err = volumeDef.Unmarshal(rawVolumeDef)

	if err != nil {
		return fmt.Errorf("RefreshVolumeDefinition.Unmarshal: %s", err)
	}

	pctx.VolumeDefinition = volumeDef
	return nil
}

func (pctx *PreparationContext) uploadVolume(path string) multistep.StepAction {

	fPtr, err := os.Open(path)
	if err != nil {
		pctx.HaltOnError(err, "UploadVolume.Open: %s", err)
	}

	defer fPtr.Close()

	fInfo, err := fPtr.Stat()
	if err != nil {
		pctx.HaltOnError(err, "UploadVolume.Stat: %s", err)
	}

	size := uint64(fInfo.Size())
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

	err = pctx.Driver.StorageVolUpload(*pctx.VolumeRef, fPtr, 0, size, 0)

	if err != nil {
		connectUri, _ := pctx.Driver.ConnectGetUri()

		// The test backend does not support Volume Uploads, so
		if connectUri[0:4] == "test" {
			pctx.Ui.Error(fmt.Sprintf("UploadVolume.Upload: %s", err))
		} else {
			return pctx.HaltOnError(err, "UploadVolume.Upload: %s", err)
		}
	}

	err = pctx.RefreshVolumeDefinition()

	if err != nil {
		log.Printf("Error while refreshing volume definition: %s\n", err)
	}

	return multistep.ActionContinue
}

func (pctx *PreparationContext) HaltOnError(err error, s string, a ...interface{}) multistep.StepAction {
	err2 := fmt.Errorf(s, a...)
	pctx.State.Put("error", err2)
	pctx.Ui.Error(err2.Error())
	return multistep.ActionHalt
}
