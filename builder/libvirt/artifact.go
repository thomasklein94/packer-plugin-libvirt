package libvirt

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

type Artifact struct {
	volumeDef     libvirtxml.StorageVolume
	volumeRef     libvirt.StorageVol
	driver        *libvirt.Libvirt
	generatedData map[string]interface{}
}

// Returns the ID of the builder that was used to create this artifact.
// This is the internal ID of the builder and should be unique to every
// builder. This can be used to identify what the contents of the
// artifact actually are.
func (artifact *Artifact) BuilderId() string {
	return builderId
}

// Returns the set of files that comprise this artifact. If an
// artifact is not made up of files, then this will be empty.

func (artifact *Artifact) Files() []string {
	return []string{}
}

// The ID for the artifact, if it has one. This is not guaranteed to
// be unique every run (like a GUID), but simply provide an identifier
// for the artifact that may be meaningful in some way. For example,
// for Amazon EC2, this value might be the AMI ID.
func (artifact *Artifact) Id() string {
	return artifact.volumeDef.Key
}

// The ID for the artifact, if it has one. This is not guaranteed to
// be unique every run (like a GUID), but simply provide an identifier
// for the artifact that may be meaningful in some way. For example,
// for Amazon EC2, this value might be the AMI ID.

func (artifact *Artifact) String() string {
	return fmt.Sprintf(
		"Libvirt volume %s/%s in %s format was generated",
		artifact.volumeRef.Pool,
		artifact.volumeDef.Name,
		artifact.volumeDef.Target.Format.Type,
	)
}

// State allows the caller to ask for builder specific state information
// relating to the artifact instance.
func (artifact *Artifact) State(name string) interface{} {
	switch name {
	case "Key":
		return artifact.volumeDef.Key
	case "Pool":
		return artifact.volumeRef.Pool
	case "Volume":
		return artifact.volumeDef.Name
	case "Allocation":
		return fmt.Sprintf("%d%s", artifact.volumeDef.Allocation.Value, artifact.volumeDef.Allocation.Unit)
	case "Size", "Capacity":
		return fmt.Sprintf("%d%s", artifact.volumeDef.Capacity.Value, artifact.volumeDef.Capacity.Unit)
	case "Physical":
		return fmt.Sprintf("%d%s", artifact.volumeDef.Physical.Value, artifact.volumeDef.Physical.Unit)
	case "Format":
		return artifact.volumeDef.Target.Format.Type
	case "RemotePath":
		return artifact.volumeDef.Target.Path
	default:
		if v, ok := artifact.generatedData[name]; ok {
			return v
		}
	}
	return nil
}

// Destroy deletes the artifact. Packer calls this for various reasons,
// such as if a post-processor has processed this artifact and it is
// no longer needed.
func (artifact *Artifact) Destroy() error {
	err := artifact.driver.StorageVolDelete(artifact.volumeRef, libvirt.StorageVolDeleteNormal)
	return err
}
