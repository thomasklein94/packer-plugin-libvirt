package libvirt

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	libvirtutils "github.com/thomasklein94/packer-plugin-libvirt/libvirt-utils"
)

type stepStartDomain struct{}

func (s *stepStartDomain) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(*libvirt.Libvirt)
	domain := state.Get("domain").(*libvirt.Domain)

	ui.Say("Starting the Libvirt domain")

	err := driver.DomainCreate(*domain)

	if err != nil {
		return haltOnError(ui, state, "DomainCreate.RPC: %s", err)
	}

	if config.PackerDebug {
		if consoleAlias := os.Getenv("PACKER_LIBVIRT_STREAM_CONSOLE"); consoleAlias != "" {
			consoleAlias = fmt.Sprintf("ua-%s", consoleAlias)
			go streamConsole(driver, *domain, consoleAlias, log.Writer())
		}
	}

	return multistep.ActionContinue
}

func (s *stepStartDomain) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(*libvirt.Libvirt)
	domain := state.Get("domain").(*libvirt.Domain)
	config := state.Get("config").(*Config)

	domainState, _, err := driver.DomainGetState(*domain, 0)

	if libvirtutils.DomainStateMeansStopped(libvirt.DomainState(domainState)) {
		log.Println("stepStartDomain.Cleanup: domain already stopped")
		return
	}

	if _, ok := state.GetOk("shutdown_sent"); ok {
		log.Println("stepStartDomain.Cleanup: shutdown already sent, skipping graceful shutdown")
	} else {
		ctx := context.TODO()

		subCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		pollErrs := make(chan error, 1)
		pollResults := make(chan libvirt.DomainState)
		timeout := time.After(config.ShutdownTimeout)
		period := 5 * time.Second

		go libvirtutils.PollDomainState(subCtx, period, *driver, *domain, pollResults, pollErrs)

	checks:
		for {
			select {
			case <-timeout:
				cancel()
				ui.Error("Domain did not stopped in time")
				break checks

			case res := <-pollResults:
				if libvirtutils.DomainStateMeansStopped(res) {
					if res == libvirt.DomainCrashed {
						ui.Error("Domain crashed while waiting for a graceful shutdown")
						break checks
					}
					ui.Say("Domain gracefully stopped")
					break checks
				}

			case err := <-pollErrs:
				ui.Error(fmt.Sprintf("Error while waiting for a clean shutdown: %s", err))
				break checks

			case <-ctx.Done():
				log.Printf("stepStartdownDomain.Cleanup: %s", err)
				cancel()
				break checks
			}
		}
	}

	driver.DomainDestroy(*domain)
}

func streamConsole(driver *libvirt.Libvirt, dom libvirt.Domain, devAlias string, stream io.Writer) error {
	err := driver.DomainOpenConsole(dom, libvirt.OptString{devAlias}, stream, uint32(libvirt.DomainConsoleForce))

	if err != nil {
		return err
	}
	return nil
}
