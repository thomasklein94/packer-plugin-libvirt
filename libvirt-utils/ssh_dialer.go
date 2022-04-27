package libvirtutils

import (
	"fmt"
	"io/ioutil"
	"log"
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
	log.Println("Libvirt SSH transport connected")

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

	key_path, ok := uri.GetExtra(LibvirtUriParam_Keyfile)
	if !ok {
		return fmt.Errorf("ssh transport requires %s parameter", LibvirtUriParam_Keyfile)
	}

	expanded_key_path, err := pathing.ExpandUser(key_path)

	if err != nil {
		return err
	}

	key, err := ioutil.ReadFile(expanded_key_path)

	if err != nil {
		return err
	}

	parsed_key, err := ssh.ParsePrivateKey(key)

	if err != nil {
		return err
	}

	dialer.sshConfig.Auth = append(dialer.sshConfig.Auth, ssh.PublicKeys(parsed_key))

	return
}

func sshDialerSetVerification(uri LibvirtUri, dialer *SshDialer) error {
	no_verify := false

	if no_verify_string, ok := uri.GetExtra(LibvirtUriParam_NoVerify); ok {
		no_verify_num, err := strconv.Atoi(no_verify_string)
		if err != nil {
			return err
		}
		no_verify = no_verify_num > 0
	}

	if no_verify {
		dialer.sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		return nil
	}

	known_host_path, ok := uri.GetExtra(LibvirtUriParam_KnownHost)

	if !ok {
		return fmt.Errorf("either %s=1 must be specified, or %s must be a path", LibvirtUriParam_NoVerify, LibvirtUriParam_KnownHost)
	}

	host_key_callback, err := knownhosts.New(known_host_path)
	if err != nil {
		return err
	}

	dialer.sshConfig.HostKeyCallback = host_key_callback
	return nil
}
