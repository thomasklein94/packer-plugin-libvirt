package libvirtutils

import "github.com/digitalocean/go-libvirt/socket/dialers"

func NewUnixDialer(uri LibvirtUri) *dialers.Local {
	opts := []dialers.LocalOption{}

	if pathOverride, ok := uri.GetExtra(LibvirtUriParam_Socket); ok {
		opts = append(opts, dialers.WithSocket(pathOverride))
	}

	return dialers.NewLocal(opts...)
}
