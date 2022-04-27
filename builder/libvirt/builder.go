package libvirt

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	libvirtutils "github.com/thomasklein94/packer-plugin-libvirt/libvirt-utils"
)

const builderId = "thomasklein94.libvirt"

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	warnings, errs := b.config.Prepare(raws...)

	return nil, warnings, errs
}

func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	var artifact *Artifact = nil

	driver, err := libvirtutils.ConnectByUriString(b.config.LibvirtURI)
	if err != nil {
		return nil, err
	}

	domainDef := newDomainDefinition(&b.config)

	// Setup the state bag
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("domain_def", &domainDef)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)

	steps := []multistep.Step{}
	steps = append(steps,
		&stepPrepareVolumes{},
		&stepCreateDomain{},
	)

	switch b.config.Communicator.Type {
	case "ssh":
		steps = append(steps, &communicator.StepConnectSSH{
			Config:    &b.config.Communicator,
			Host:      GetDomainCommunicatorAddress,
			SSHConfig: b.config.Communicator.SSHConfigFunc(),
		})
	case "winrm":
		steps = append(steps, &communicator.StepConnectWinRM{
			Config: &b.config.Communicator,
			Host:   GetDomainCommunicatorAddress,
		})
	default:
		ui.Error(fmt.Sprintf("Unsupported communicator type '%s', no communication will be established to domain", b.config.Communicator.Type))
	}

	steps = append(steps, &commonsteps.StepProvision{})

	// Run
	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("build was cancelled")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("build was halted")
	}

	if _, ok := state.GetOk("artifact"); ok {
		artifact = state.Get("artifact").(*Artifact)
		artifact.generatedData = map[string]interface{}{}
	}

	return artifact, nil
}

func haltOnError(ui packersdk.Ui, state multistep.StateBag, s string, a ...interface{}) multistep.StepAction {
	err := fmt.Errorf(s, a...)
	state.Put("error", err)
	ui.Error(err.Error())
	return multistep.ActionHalt
}
