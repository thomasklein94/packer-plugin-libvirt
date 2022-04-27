package libvirtutils

import (
	"fmt"

	"github.com/digitalocean/go-libvirt/socket/dialers"
)

func NewTcpDialer(uri LibvirtUri) (*dialers.Remote, error) {
	opts := []dialers.RemoteOption{}

	if uri.Hostname == "" {
		return nil, fmt.Errorf("hostname must be specified for tcp transport")
	}

	port := uri.Port
	if port == "" {
		port = "16509"
	}
	address := fmt.Sprintf("%s:%s", uri.Hostname, uri.Port)

	return dialers.NewRemote(address, opts...), nil
}
