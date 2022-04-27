package libvirtutils

import (
	"fmt"
	"log"

	"github.com/digitalocean/go-libvirt"
	libvirt_socket "github.com/digitalocean/go-libvirt/socket"
)

func NewDialerFromLibvirtUri(uri LibvirtUri) (dialer libvirt_socket.Dialer, err error) {
	switch uri.Transport {
	case "ssh":
		dialer, err = NewSshDialer(uri)
	case "tls":
		dialer, err = newTlsDialer(uri)
	case "tcp":
		dialer, err = NewTcpDialer(uri)
	case "unix", "":
		dialer = NewUnixDialer(uri)
	}

	return
}

func ConnectByUriString(libvirt_uri string) (*libvirt.Libvirt, error) {
	uri := LibvirtUri{}
	err := uri.Unmarshal(libvirt_uri)
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

	name := uri.Name()
	log.Printf("[DEBUG] Sending '%s' to libvirtd as URI\n", name)
	err = connection.ConnectToURI(libvirt.ConnectURI(name))

	if err != nil {
		if uri.Driver == "test" {
			err = nil
		} else {
			err = fmt.Errorf("error while establishing connection with libvirt daemon: %s", err)
		}
	}
	return connection, err
}
