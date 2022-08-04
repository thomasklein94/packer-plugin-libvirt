package libvirtutils

import (
	"fmt"

	"github.com/digitalocean/go-libvirt/socket/dialers"
)

func NewTcpDialer(uri LibvirtUri) (*dialers.Remote, error) {
	if uri.Hostname == "" {
		return nil, fmt.Errorf("hostname must be specified for tcp transport")
	}

	var opts []dialers.RemoteOption
	if uri.Port != "" {
		opts = append(opts, dialers.UsePort(uri.Port))
	}

	return dialers.NewRemote(uri.Hostname, opts...), nil
}
