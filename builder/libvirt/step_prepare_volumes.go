package libvirt

import (
	"context"
	"fmt"
	"log"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/thomasklein94/packer-plugin-libvirt/builder/libvirt/volume"
	"libvirt.org/go/libvirtxml"
)

type stepPrepareVolumes struct {
	preparations []*volume.PreparationContext
}

func (s *stepPrepareVolumes) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.preparations = []*volume.PreparationContext{}

	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(*libvirt.Libvirt)
	config := state.Get("config").(*Config)
	domainDef := state.Get("domain_def").(*libvirtxml.Domain)

	ui.Say("Preparing volumes...")

	for _, volumeConfig := range config.Volumes {
		pctx := &volume.PreparationContext{
			State:            state,
			Ui:               ui,
			Driver:           driver,
			VolumeConfig:     nil,
			VolumeRef:        nil,
			VolumeDefinition: nil,
			PoolRef:          nil,
			VolumeIsCreated:  false,
			VolumeIsArtifact: volumeConfig.Alias == config.ArtifactVolumeAlias,
			Context:          ctx,
		}

		s.preparations = append(s.preparations, pctx)
		if action := volumeConfig.PrepareVolume(pctx); action != multistep.ActionContinue {
			return action
		}

		domainDisk := volumeConfig.DomainDiskXml()
		if domainDisk != nil {
			domainDef.Devices.Disks = append(domainDef.Devices.Disks, *domainDisk)
		}

	}

	return multistep.ActionContinue
}

func (s *stepPrepareVolumes) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)
	ui.Say("Cleaning up volumes...")

	for _, pctx := range s.preparations {
		log.Printf("Checking volume %s/%s for cleanup\n", pctx.VolumeConfig.Pool, pctx.VolumeConfig.Name)
		if pctx.VolumeRef != nil && pctx.VolumeIsCreated {
			abort, abortSet := state.GetOk("aborted")
			_, canceled := state.GetOk(multistep.StateCancelled)
			_, halted := state.GetOk(multistep.StateHalted)

			delete := !pctx.VolumeIsArtifact || (abortSet && abort.(bool)) || canceled || halted

			if delete {
				pctx.Ui.Message(fmt.Sprintf("Cleaning up volume %s/%s", pctx.VolumeRef.Pool, pctx.VolumeRef.Name))
				err := pctx.Driver.StorageVolDelete(*pctx.VolumeRef, libvirt.StorageVolDeleteNormal)
				if err != nil {
					pctx.Ui.Error(fmt.Sprintf("Couldn't clean up volume %s/%s: %s", pctx.VolumeConfig.Pool, pctx.VolumeConfig.Name, err))
				}
			} else {
				if pctx.VolumeIsArtifact {
					pctx.RefreshVolumeDefinition()
					state.Put("artifact", &Artifact{
						volumeDef: *pctx.VolumeDefinition,
						volumeRef: *pctx.VolumeRef,
						driver:    pctx.Driver,
					})
				}
			}
		}
	}
}
