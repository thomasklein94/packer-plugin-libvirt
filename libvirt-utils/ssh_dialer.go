package libvirtutils

import (
	"fmt"
	"io/ioutil"
	"net"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/pathing"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SshDialer struct {
	address          string
	sshConfig        *ssh.ClientConfig
	sshClient        *ssh.Client
	remoteUnixSocket string
}

func (dialer *SshDialer) Dial() (net.Conn, error) {
	client, err := ssh.Dial("tcp", dialer.address, dialer.sshConfig)
	if err != nil {
		return nil, fmt.Errorf("error connecting to libvirt via ssh: %s", err)
	}
	dialer.sshClient = client

	return client.Dial("unix", dialer.remoteUnixSocket)
}

func NewSshDialer(uri LibvirtUri) (dialer *SshDialer, err error) {
	dialer = &SshDialer{
		sshConfig: &ssh.ClientConfig{
			Auth:    []ssh.AuthMethod{},
			Timeout: 0,
		},
		sshClient: nil,
	}

	if err = sshSetAddress(uri, dialer); err != nil {
		return
	}
	if err = sshDialerSetPrivateKey(uri, dialer); err != nil {
		return
	}
	if err = sshDialerSetVerification(uri, dialer); err != nil {
		return
	}
	if err = sshSetSshUser(uri, dialer); err != nil {
		return
	}
	if err = sshSetRemoteUnixSocket(uri, dialer); err != nil {
		return
	}

	return dialer, nil
}

func sshSetSshUser(uri LibvirtUri, dialer *SshDialer) error {
	if uri.Username == "" {
		return fmt.Errorf("username must be specified for ssh transport")
	}
	dialer.sshConfig.User = uri.Username
	return nil
}

func sshSetRemoteUnixSocket(uri LibvirtUri, dialer *SshDialer) error {
	var ok bool
	if dialer.remoteUnixSocket, ok = uri.GetExtra(LibvirtUriParam_Socket); !ok {
		dialer.remoteUnixSocket = "/var/run/libvirt/libvirt-sock"
	}

	return nil
}

func sshSetAddress(uri LibvirtUri, dialer *SshDialer) error {
	if uri.Hostname == "" {
		return fmt.Errorf("hostname must be specified for ssh transport")
	}
	port := uri.Port

	if port == "" {
		port = "22"
	}

	dialer.address = fmt.Sprintf("%s:%s", uri.Hostname, port)

	return nil
}

func sshDialerSetPrivateKey(uri LibvirtUri, dialer *SshDialer) (err error) {

	keyPath, ok := uri.GetExtra(LibvirtUriParam_Keyfile)
	if !ok {
		return fmt.Errorf("ssh transport requires %s parameter", LibvirtUriParam_Keyfile)
	}

	expandedKeyPath, err := pathing.ExpandUser(keyPath)

	if err != nil {
		return err
	}

	key, err := ioutil.ReadFile(expandedKeyPath)

	if err != nil {
		return err
	}

	parsedKey, err := ssh.ParsePrivateKey(key)

	if err != nil {
		return err
	}

	dialer.sshConfig.Auth = append(dialer.sshConfig.Auth, ssh.PublicKeys(parsedKey))

	return
}

func sshDialerSetVerification(uri LibvirtUri, dialer *SshDialer) error {
	noVerify := false

	if noVerifyString, ok := uri.GetExtra(LibvirtUriParam_NoVerify); ok {
		noVerifyNum, err := strconv.Atoi(noVerifyString)
		if err != nil {
			return err
		}
		noVerify = noVerifyNum != 0
	}

	if noVerify {
		dialer.sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		return nil
	}

	knownHostPath, ok := uri.GetExtra(LibvirtUriParam_KnownHost)

	if !ok {
		return fmt.Errorf("either %s=1 must be specified, or %s must be a path", LibvirtUriParam_NoVerify, LibvirtUriParam_KnownHost)
	}

	expandedKnownHostPath, err := pathing.ExpandUser(knownHostPath)

	if err != nil {
		return err
	}

	hostKeyCallback, err := knownhosts.New(expandedKnownHostPath)
	if err != nil {
		return err
	}

	dialer.sshConfig.HostKeyCallback = hostKeyCallback
	return nil
}
