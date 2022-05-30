package libvirt

import (
	"context"
	"fmt"
	"log"
	"time"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	libvirtutils "github.com/thomasklein94/packer-plugin-libvirt/libvirt-utils"
)

type stepShutdownDomain struct {
}

func (s *stepShutdownDomain) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(*libvirt.Libvirt)
	domain := state.Get("domain").(*libvirt.Domain)

	ui.Say("Shutting down libvirt domain...")
	shutdownMode, _ := mapDomainShutdown(config.ShutdownMode)

	err := driver.DomainShutdownFlags(*domain, shutdownMode)
	if err != nil {
		fmterr := fmt.Errorf("couldn't shut down domain gracefully: %s", err)
		ui.Error(fmterr.Error())

		return multistep.ActionHalt
	}

	state.Put("shutdown_sent", true)

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	pollErrs := make(chan error, 1)
	pollResults := make(chan libvirt.DomainState)
	timeout := time.After(config.ShutdownTimeout)
	period := 5 * time.Second

	go libvirtutils.PollDomainState(subCtx, period, *driver, *domain, pollResults, pollErrs)

	for {
		select {
		case <-timeout:
			cancel()
			ui.Error("Domain did not stopped in time")
			return multistep.ActionHalt

		case res := <-pollResults:
			if libvirtutils.DomainStateMeansStopped(res) {
				if res == libvirt.DomainCrashed {
					ui.Error("Domain crashed while waiting for a graceful shutdown")
					return multistep.ActionHalt
				}
				ui.Say("Domain gracefully stopped")
				return multistep.ActionContinue
			}

		case err := <-pollErrs:
			cancel()
			return haltOnError(ui, state, "error while waiting for a clean shutdown: %s", err)

		case <-ctx.Done():
			log.Printf("stepShutdownDomain: %s", err)
			cancel()
			return multistep.ActionHalt
		}
	}
}

func (s *stepShutdownDomain) Cleanup(state multistep.StateBag) {
}
