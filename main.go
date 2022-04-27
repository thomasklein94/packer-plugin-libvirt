package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/plugin"

	"github.com/thomasklein94/packer-plugin-libvirt/builder/libvirt"
	"github.com/thomasklein94/packer-plugin-libvirt/version"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder(plugin.DEFAULT_NAME, new(libvirt.Builder))
	pps.SetVersion(version.PluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
