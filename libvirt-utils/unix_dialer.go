package libvirtutils

import "github.com/digitalocean/go-libvirt/socket/dialers"

func NewUnixDialer(uri LibvirtUri) (*dialers.Local) {
	opts := []dialers.LocalOption{}
	
	if unix_path_override, ok := uri.GetExtra(LibvirtUriParam_Socket); ok {
		opts = append(opts, dialers.WithSocket(unix_path_override))
	}

	return dialers.NewLocal(opts...)
}
