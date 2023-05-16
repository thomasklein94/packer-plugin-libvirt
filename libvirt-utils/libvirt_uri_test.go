package libvirtutils_test

import (
	"reflect"
	"testing"

	libvirtutils "github.com/thomasklein94/packer-plugin-libvirt/libvirt-utils"
)

func TestValidUriUnmarshall(t *testing.T) {
	expectations := map[string]libvirtutils.LibvirtUri{
		"qemu:///system": {
			Driver:      "qemu",
			Transport:   "",
			Username:    "",
			Hostname:    "",
			Port:        "",
			Path:        "/system",
			ExtraParams: map[string]string{},
		},

		"qemu+ssh://someuser@somehost:8022/system": {
			Driver:      "qemu",
			Transport:   "ssh",
			Username:    "someuser",
			Hostname:    "somehost",
			Port:        "8022",
			Path:        "/system",
			ExtraParams: map[string]string{},
		},

		"qemu+ssh://someuser@somehost:8022/system?keyfile=/path/to/key": {
			Driver:    "qemu",
			Transport: "ssh",
			Username:  "someuser",
			Hostname:  "somehost",
			Port:      "8022",
			Path:      "/system",
			ExtraParams: map[string]string{
				"keyfile": "/path/to/key",
			},
		},

		"qemu+ssh://someuser@somehost:8022/system?keyfile=/path/to/key&no_verify=1": {
			Driver:    "qemu",
			Transport: "ssh",
			Username:  "someuser",
			Hostname:  "somehost",
			Port:      "8022",
			Path:      "/system",
			ExtraParams: map[string]string{
				"no_verify": "1",
				"keyfile":   "/path/to/key",
			},
		},

		"qemu+ssh://s0m3us3r@some.host:8022/system?keyfile=/path/to/key&no_verify=1": {
			Driver:    "qemu",
			Transport: "ssh",
			Username:  "s0m3us3r",
			Hostname:  "some.host",
			Port:      "8022",
			Path:      "/system",
			ExtraParams: map[string]string{
				"no_verify": "1",
				"keyfile":   "/path/to/key",
			},
		},

		"qemu+ssh://s0m3us3r$@so.me.ho.st:8022/system": {
			Driver:      "qemu",
			Transport:   "ssh",
			Username:    "s0m3us3r$",
			Hostname:    "so.me.ho.st",
			Port:        "8022",
			Path:        "/system",
			ExtraParams: map[string]string{},
		},
	}

	for raw, expected := range expectations {
		parsed := libvirtutils.LibvirtUri{}
		err := parsed.Unmarshal(raw)

		if err != nil {
			t.Fatalf("Unexpected error while unmarshalling '%s': %s", raw, err)
		}

		if !reflect.DeepEqual(parsed, expected) {
			t.Fatalf("%s Unmarshalled to %+v, expected %+v", raw, parsed, expected)
		}
	}
}

func TestValidTestUriUnmarshall(t *testing.T) {
	expectations := map[string]libvirtutils.LibvirtUri{
		"test:///default": {
			Driver:      "test",
			Transport:   "",
			Username:    "",
			Hostname:    "",
			Port:        "",
			Path:        "/default",
			ExtraParams: map[string]string{},
		},

		"test:///path/to/config.xml": {
			Driver:      "test",
			Transport:   "",
			Username:    "",
			Hostname:    "",
			Port:        "",
			Path:        "/path/to/config.xml",
			ExtraParams: map[string]string{},
		},
	}

	for raw, expected := range expectations {
		parsed := libvirtutils.LibvirtUri{}
		err := parsed.Unmarshal(raw)

		if err != nil {
			t.Fatalf("Unexpected error while unmarshalling '%s': %s", raw, err)
		}

		if !reflect.DeepEqual(parsed, expected) {
			t.Fatalf("%s Unmarshalled to %+v, expected %+v", raw, parsed, expected)
		}
	}
}
