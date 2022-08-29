package libvirtutils

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket"
)

func NewDialerFromLibvirtUri(uri LibvirtUri) (dialer socket.Dialer, err error) {
	switch uri.Transport {
	case "ssh":
		dialer, err = NewSshDialer(uri)
	case "tls":
		dialer, err = NewTlsDialer(uri)
	case "tcp":
		dialer, err = NewTcpDialer(uri)
	case "unix", "":
		dialer = NewUnixDialer(uri)
	default:
		err = fmt.Errorf("%s is not supported uri transport", uri.Transport)
	}

	return
}

func ConnectByUriString(libvirtUri string) (*libvirt.Libvirt, error) {
	uri := LibvirtUri{}
	err := uri.Unmarshal(libvirtUri)
	if err != nil {
		return nil, err
	}
	return ConnectByUri(uri)
}

func ConnectByUri(uri LibvirtUri) (*libvirt.Libvirt, error) {
	dialer, err := NewDialerFromLibvirtUri(uri)
	if err != nil {
		return nil, err
	}

	connection := libvirt.NewWithDialer(dialer)

	err = connection.ConnectToURI(libvirt.ConnectURI(uri.Name()))
	if err != nil {
		if uri.Driver == "test" {
			err = nil
		} else {
			err = fmt.Errorf("error while establishing connection with libvirt daemon: %s", err)
		}
	}
	return connection, err
}
