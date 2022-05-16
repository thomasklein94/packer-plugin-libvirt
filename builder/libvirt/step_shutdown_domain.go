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
	shutdown_mode, _ := mapDomainShutdown(config.ShutdownMode)

	err := driver.DomainShutdownFlags(*domain, shutdown_mode)
	if err != nil {
		fmterr := fmt.Errorf("couldn't shut down domain gracefully: %s", err)
		ui.Error(fmterr.Error())

		return multistep.ActionHalt
	}

	state.Put("shutdown_sent", true)

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	poll_errs := make(chan error, 1)
	poll_results := make(chan libvirt.DomainState)
	timeout := time.After(config.ShutdownTimeout)
	period := 5 * time.Second

	go libvirtutils.PollDomainState(subCtx, period, *driver, *domain, poll_results, poll_errs)

	for {
		select {
		case <-timeout:
			cancel()
			ui.Error("Domain did not stopped in time")
			return multistep.ActionHalt

		case res := <-poll_results:
			if libvirtutils.DomainStateMeansStopped(res) {
				if res == libvirt.DomainCrashed {
					ui.Error("Domain crashed while waiting for a graceful shutdown")
					return multistep.ActionHalt
				}
				ui.Say("Domain gracefully stopped")
				return multistep.ActionContinue
			}

		case err := <-poll_errs:
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
