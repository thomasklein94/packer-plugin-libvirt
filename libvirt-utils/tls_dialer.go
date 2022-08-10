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
	address   string
	tlsConfig tls.Config
}

const (
	pkiCACert     = "cacert.pem"
	pkiClientCert = "clientcert.pem"
	pkiClientKey  = "clientkey.pem"
)

func (dialer *TlsDialer) Dial() (net.Conn, error) {
	conn, err := tls.Dial("tcp", dialer.address, &dialer.tlsConfig)
	// Workaround for hanging TLS connection described here: https://github.com/digitalocean/go-libvirt/issues/89
	if err == nil {
		_, err = conn.Read(make([]byte, 1))
		if err != nil {
			return nil, err
		}
	}
	return conn, err
}

func NewTlsDialer(uri LibvirtUri) (dialer *TlsDialer, err error) {
	dialer = &TlsDialer{
		address:   "",
		tlsConfig: tls.Config{},
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
	noVerify := false

	if noVerifyString, ok := uri.GetExtra(LibvirtUriParam_NoVerify); ok {
		noVerifyNum, err := strconv.Atoi(noVerifyString)
		if err != nil {
			return err
		}
		noVerify = noVerifyNum > 0
	}

	dialer.tlsConfig.InsecureSkipVerify = noVerify

	if noVerify {
		return nil
	}

	pkiPath, ok := uri.GetExtra(LibvirtUriParam_PkiPath)

	if !ok {
		return fmt.Errorf("either %s=1 must be specified, or %s must be a path to a directory", LibvirtUriParam_NoVerify, LibvirtUriParam_PkiPath)
	}

	// CAcert
	caCertPath := filepath.Join(pkiPath, pkiCACert)
	caCert, err := ioutil.ReadFile(caCertPath)

	if err != nil {
		return err
	}

	dialer.tlsConfig.RootCAs = x509.NewCertPool()
	dialer.tlsConfig.RootCAs.AppendCertsFromPEM(caCert)

	// Client cert and key

	clientCertPath := filepath.Join(pkiPath, pkiClientCert)
	keyfilePath := filepath.Join(pkiPath, pkiClientKey)
	keypair, err := tls.LoadX509KeyPair(clientCertPath, keyfilePath)

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
