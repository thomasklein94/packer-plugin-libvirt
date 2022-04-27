package libvirtutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
)

type TlsDialer struct {
	address          string
	tlsConfig        tls.Config
}

const (
	Pki_CaCert     = "cacert.pem"
	Pki_ClientCert = "clientcert.pem"
	Pki_ClientKey  = "clientkey.pem"
)

func (dialer *TlsDialer) Dial() (net.Conn, error) {
	return tls.Dial("tcp", dialer.address, &dialer.tlsConfig)
}

func newTlsDialer(uri LibvirtUri) (dialer *TlsDialer, err error) {
	dialer = &TlsDialer{
		address:          "",
		tlsConfig:        tls.Config{},
	}
	if err = tlsSetAddress(uri, dialer); err != nil {
		return
	}
	if err = tlsDialerSetVerification(uri, dialer); err != nil {
		return
	}

	return dialer, nil
}

func tlsDialerSetVerification(uri LibvirtUri, dialer *TlsDialer) error {
	no_verify := false

	if no_verify_string, ok := uri.GetExtra(LibvirtUriParam_NoVerify); ok {
		no_verify_num, err := strconv.Atoi(no_verify_string)
		if err != nil {
			return err
		}
		no_verify = no_verify_num > 0
	}

	dialer.tlsConfig.InsecureSkipVerify = no_verify

	if no_verify {
		return nil
	}

	pki_path, ok := uri.GetExtra(LibvirtUriParam_PkiPath)

	if !ok {
		return fmt.Errorf("either %s=1 must be specified, or %s must be a path to a directory", LibvirtUriParam_NoVerify, LibvirtUriParam_PkiPath)
	}

	// CAcert
	cacert_path := filepath.Join(pki_path, Pki_CaCert)
	cacert, err := ioutil.ReadFile(cacert_path)

	if err != nil {
		return err
	}

	dialer.tlsConfig.RootCAs = x509.NewCertPool()
	dialer.tlsConfig.RootCAs.AppendCertsFromPEM(cacert)

	// Client cert and key

	clientcert_path := filepath.Join(pki_path, Pki_ClientCert)
	keyfile_path := filepath.Join(pki_path, Pki_ClientKey)
	keypair, err := tls.LoadX509KeyPair(clientcert_path, keyfile_path)

	if err != nil {
		return err
	}

	dialer.tlsConfig.Certificates = []tls.Certificate{keypair}

	return nil
}

func tlsSetAddress(uri LibvirtUri, dialer *TlsDialer) error {
	if uri.Hostname == "" {
		return fmt.Errorf("hostname must be specified for tls transport")
	}
	port := uri.Port

	if port == "" {
		port = "16514"
	}

	dialer.address = fmt.Sprintf("%s:%s", uri.Hostname, port)

	return nil
}
