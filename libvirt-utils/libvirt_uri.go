package libvirtutils

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

// https://libvirt.org/uri.html
// driver[+transport]://[username@][hostname][:port]/[path][?extraparameters]
type LibvirtUri struct {
	Driver    string
	Transport string
	Username  string
	Hostname  string
	Port      string
	Path      string
	// https://libvirt.org/uri.html#Remote_URI_parameters
	ExtraParams map[string]string
}

type LibvirtUriExtraParam string

// A copy of https://libvirt.org/uri.html#Remote_URI_parameters.
// Only a handful of these implemented yet, and not all key or option might make sense in this setup
const (
	// (Any transport) The name passed to the remote virConnectOpen function.
	// The name is normally formed by removing transport, hostname, port number, username and extra parameters from the remote URI,
	// but in certain very complex cases it may be better to supply the name explicitly.
	LibvirtUriParam_Name LibvirtUriExtraParam = "name"
	// (TLS transport only) A valid GNUTLS priority string
	LibvirtUriParam_TlsPriority LibvirtUriExtraParam = "tls_priority"

	LibvirtUriParam_Mode   LibvirtUriExtraParam = "mode"
	LibvirtUriParam_Socket LibvirtUriExtraParam = "socket"
	// (SSH transport only)  The name of the private key file to use to authentication to the remote machine.
	LibvirtUriParam_Keyfile LibvirtUriExtraParam = "keyfile"
	// SSH: If set to a non-zero value, this disables client's strict host key checking making it auto-accept new host keys. Existing host keys will still be validated.
	// TLS: If set to a non-zero value, this disables client checks of the server's certificate. Note that to disable server checks of the client's certificate or IP address you must change the libvirtd configuration.
	LibvirtUriParam_NoVerify LibvirtUriExtraParam = "no_verify"
	// Path to the known_hosts file to verify the host key against.
	LibvirtUriParam_KnownHost LibvirtUriExtraParam = "known_hosts"
	// Specifies x509 certificates path for the client. If any of the CA certificate, client certificate, or client key is missing, the connection will fail with a fatal error.
	LibvirtUriParam_PkiPath LibvirtUriExtraParam = "pkipath"
)

func (uri *LibvirtUri) GetExtra(p LibvirtUriExtraParam) (string, bool) {
	val, ok := uri.ExtraParams[string(p)]
	return val, ok
}

func (uri *LibvirtUri) Unmarshal(s string) error {
	uriRegex := `^(?P<Driver>[a-z]+)(\+(?P<Transport>[a-z]+))?://(((?P<Username>[a-z_][-a-z0-9_]*\$?)@)?(?P<Hostname>[-_.a-z0-9]+)(:(?P<Port>[0-9]+)?)?)?(?P<Path>/[-_.a-z0-9]+)?(\?(?P<extra>.*))?$`
	re := regexp.MustCompile(uriRegex)

	matches := allMatchedRegexpGroups(re, s)

	if len(matches) == 0 {
		testUriRegex := `^(?P<Driver>test)://(?P<Username>)(?P<Hostname>)(?P<Port>)(?P<Path>(default|/[-_.a-z0-9/]+))?(\?(?P<extra>.*))?$`
		re = regexp.MustCompile(testUriRegex)

		matches = allMatchedRegexpGroups(re, s)

		if len(matches) == 0 {
			return fmt.Errorf("can't parse '%s' as a libvirt uri", s)
		}
	}

	uri.Driver = matches["Driver"]
	uri.Transport = matches["Transport"]
	uri.Username = matches["Username"]
	uri.Hostname = matches["Hostname"]
	uri.Port = matches["Port"]
	uri.Path = matches["Path"]
	uri.ExtraParams = map[string]string{}

	if len(matches["extra"]) > 0 {
		for _, extra := range strings.Split(matches["extra"], "&") {
			kvPair := strings.Split(extra, "=")
			if len(kvPair) != 2 {
				return fmt.Errorf("can't parse extra parameters string '%s'", extra)
			}

			uri.ExtraParams[kvPair[0]] = kvPair[1]
		}
	}

	if uri.Driver == "test" {
		if !filepath.IsAbs(uri.Path) && uri.Path != "/default" {
			abs, err := filepath.Abs(uri.Path)

			if err != nil {
				return fmt.Errorf("libvirt test driver config path translation: %s", err)
			}

			log.Printf("Libvirt test driver's configuration path changed from '%s' to '%s'\n", uri.Path, abs)
			uri.Path = abs
		}
	}

	return nil
}

func (uri *LibvirtUri) Marshal() (result string) {
	result = uri.Driver
	if uri.Transport != "" {
		result += "+" + uri.Transport
	}
	result += "://"
	if uri.Username != "" {
		result += uri.Username + "@"
	}
	if uri.Hostname != "" {
		result += uri.Hostname
	}
	if uri.Port != "" {
		result += ":" + uri.Port
	}
	result += "/" + uri.Path
	if len(uri.ExtraParams) > 0 {
		result += "?"
		extras := make([]string, len(uri.ExtraParams))
		i := 0
		for k, v := range uri.ExtraParams {
			extras[i] = fmt.Sprintf("%s=%s", k, v)
			i++
		}
		result += strings.Join(extras, "&")
	}

	return
}

func (uri *LibvirtUri) Name() (result string) {
	nameParamPresent := false
	result, nameParamPresent = uri.GetExtra(LibvirtUriParam_Name)

	if !nameParamPresent {
		result = fmt.Sprintf("%s://%s", uri.Driver, uri.Path)
	}

	return
}

func allMatchedRegexpGroups(re *regexp.Regexp, s string) map[string]string {
	match := re.FindStringSubmatch(s)
	result := make(map[string]string)

	for i, val := range match {
		name := re.SubexpNames()[i]
		if name != "" {
			result[name] = val
		}
	}
	return result
}
