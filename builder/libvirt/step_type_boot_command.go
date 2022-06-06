package libvirt

import (
	"context"
	"log"
	"strings"

	"github.com/digitalocean/go-libvirt"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"golang.org/x/mobile/event/key"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepTypeBootCommand struct{}

func (s *stepTypeBootCommand) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(*libvirt.Libvirt)
	domain := state.Get("domain").(*libvirt.Domain)
	config := state.Get("config").(*Config)

	if len(config.BootConfig.BootCommand) == 0 {
		log.Printf("Skip typing boot command as boot command is empty")
	}

	var sendKeys bootcommand.SendUsbScanCodes = func(k key.Code, down bool) error {
		err := driver.DomainSendKey(
			*domain,
			uint32(libvirt.KeycodeSetUsb),
			uint32(config.BootConfig.KeyHoldType.Milliseconds()),
			[]uint32{uint32(k)},
			0,
		)

		if err != nil {
			return err
		}
		return nil
	}

	flatBootCommand := strings.Join(config.BootConfig.BootCommand, "")
	seq, err := bootcommand.GenerateExpressionSequence(flatBootCommand)

	if err != nil {
		return haltOnError(ui, state, "%s", err)
	}

	err = seq.Do(ctx, bootcommand.NewUSBDriver(sendKeys, 0))

	if err != nil {
		return haltOnError(ui, state, "%s", err)
	}

	return multistep.ActionContinue
}

func (s *stepTypeBootCommand) Cleanup(bag multistep.StateBag) {
	// Do nothing
}
