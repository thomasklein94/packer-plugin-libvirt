//go:generate go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc struct-markdown

package libvirt

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type DomainGraphic struct {
	// Type of the graphic defined with this block. Required. Must be either `vnc` or `sdl`.
	Type string `mapstructure:"type" required:"true"`

	Sdl SdlDomainGraphic `mapstructure:",squash"`
	Vnc VNCDomainGraphic `mapstructure:",squash"`
}

type SdlDomainGraphic struct {
	// An X11 Display number where the SDL window will be sent. Required for type `sdl`
	Display string `mapstructure:"display" required:"false"`
}

type VNCDomainGraphic struct {
	// TCP port used for VNC server to listen on.
	// The number zero means the hypervisor will pick a free port randomly.
	// If not set, the autoport feature will be used.
	Port int `mapstructure:"port" required:"false"`
}

func (dg *DomainGraphic) Prepare(ctx interpolate.Context) (warnings []string, errs []error) {
	switch dg.Type {
	case "sdl":
		return dg.Sdl.Prepare(ctx)
	case "vnc":
		return dg.Sdl.Prepare(ctx)
	default:
		return nil, []error{
			fmt.Errorf("unsupported domain graphics type '%s'", dg.Type),
		}
	}
}

func (dg *VNCDomainGraphic) Prepare(ctx interpolate.Context) (warnings []string, errs []error) {
	return
}

func (dg *SdlDomainGraphic) UpdateDomainGraphic(graphic *libvirtxml.DomainGraphic) {
	graphic.SDL = &libvirtxml.DomainGraphicSDL{
		Display: dg.Display,
	}
}

func (dg *VNCDomainGraphic) UpdateDomainGraphic(graphic *libvirtxml.DomainGraphic) {
	graphic.VNC = &libvirtxml.DomainGraphicVNC{}
	if dg.Port <= 0 {
		graphic.VNC.AutoPort = "true"
	} else {
		graphic.VNC.Port = dg.Port
	}
}

func (dg *SdlDomainGraphic) Prepare(ctx interpolate.Context) (warnings []string, errs []error) {
	return
}

func (dg DomainGraphic) DomainGraphic() *libvirtxml.DomainGraphic {
	graphic := &libvirtxml.DomainGraphic{}

	switch dg.Type {
	case "sdl":
		dg.Sdl.UpdateDomainGraphic(graphic)
	case "vnc":
		dg.Vnc.UpdateDomainGraphic(graphic)
	default:
		return nil
	}

	return graphic
}
