package libvirt

import (
	"time"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

const MinimalKeyHoldTime time.Duration = 5 * time.Millisecond

type BootConfig struct {
	bootcommand.BootConfig `mapstructure:",squash"`
	KeyHoldType            time.Duration `mapstructure:"key_hold_time"`
}

func (bc *BootConfig) Prepare(ctx *interpolate.Context) []error {
	errors := bc.BootConfig.Prepare(ctx)

	if bc.KeyHoldType < MinimalKeyHoldTime {
		bc.KeyHoldType = MinimalKeyHoldTime
	}

	return errors
}
