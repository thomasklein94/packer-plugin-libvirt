//go:generate go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc struct-markdown

package volume

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"libvirt.org/go/libvirtxml"
)

type filesDeviceType string

const (
	FilesUnknownDeviceType filesDeviceType = ""
	FilesCdDeviceType      filesDeviceType = "cdrom"
	FilesFloppyDeviceType  filesDeviceType = "floppy"
)

// Create an image with files local to where packer runs and make it available at boot time.
// Depending on the parent volume's `device` attribute, the result will be a cdrom or a floppy image.
// Use this volume source scarcely, for example when the files needed as early as boot time.
// In general, making local files available on the remote machine is better done with the file provisioner.
// Floppies must not exceed the size 1.44MiB.
type FilesVolumeSource struct {
	// Files can be either files or directories. An files provided here will
	// be written to the root of the CD or Floppy. Directories will be written to the
	// root of the CD as well, but will retain their subdirectory structure.
	// For a cdrom, the behaviour is the same as `cd_files`. For floppy disks, the behaviour
	// is the same as for `floppy_dirs`!
	Files []string `mapstructure:"files"`
	// Key/Value pairs to add to the CD or Floppy. The keys represent the paths, and the values
	// represent the contents. Can be used together with `files`. If any paths are specified by
	// both, `contents` will take precedence.
	Contents map[string]string `mapstructure:"contents"`
	Label    string            `mapstructure:"label"`

	filesVolumeFormat filesDeviceType `undocumented:"true"`
}

func (vs *FilesVolumeSource) PrepareConfig(ctx *interpolate.Context, vol *Volume) (warnings []string, errs []error) {
	warnings = []string{}
	errs = []error{}

	if vs.Files == nil {
		vs.Files = []string{}
	}

	vol.ReadOnly = true

	vs.filesVolumeFormat = filesDeviceType(vol.Device)

	switch vs.filesVolumeFormat {
	case FilesFloppyDeviceType:
		vol.Bus = "fdc"
	case FilesCdDeviceType:
		vol.Format = "raw"
	default:
		errs = append(errs, fmt.Errorf("local source only support cdrom or floppy devices: '%s' not supported", vs.filesVolumeFormat))
	}
	vol.allowUnspecifiedSize = true

	return
}
func (vs *FilesVolumeSource) UpdateDomainDiskXml(domainDisk *libvirtxml.DomainDisk) {
	// Everything should be already set by the volume
}
func (vs *FilesVolumeSource) UpdateStorageDefinitionXml(storageDef *libvirtxml.StorageVolume) {
	if storageDef.Target == nil {
		storageDef.Target = &libvirtxml.StorageVolumeTarget{}
	}
	if storageDef.Target.Format == nil {
		storageDef.Target.Format = &libvirtxml.StorageVolumeTargetFormat{}
	}

	switch vs.filesVolumeFormat {
	case FilesFloppyDeviceType:
		storageDef.Target.Format.Type = "raw"
	case FilesCdDeviceType:
		storageDef.Target.Format.Type = "iso"
	}
}
func (vs *FilesVolumeSource) PrepareVolume(pctx *PreparationContext) multistep.StepAction {
	var step multistep.Step
	switch vs.filesVolumeFormat {
	case FilesCdDeviceType:
		step = vs.cdromStep(pctx)
	case FilesFloppyDeviceType:
		step = vs.floppyStep(pctx)
	default:
		return pctx.HaltOnError(nil, "Unkown volume device type: %s", vs.filesVolumeFormat)
	}

	tempState := &multistep.BasicStateBag{}
	tempState.Put("ui", pctx.Ui)

	action := step.Run(pctx.Context, tempState)
	if action != multistep.ActionContinue {
		err := fmt.Errorf("unknown error")
		rawStepError, ok := tempState.GetOk("error")
		if ok {
			err = rawStepError.(error)
		}
		return pctx.HaltOnError(err, "Couldn't assemble volume: %s", err)
	}

	path, err := vs.path(tempState)

	if err != nil {
		return pctx.HaltOnError(err, "couldn't get path for the created image")
	}

	log.Printf("Image was created at '%s'\n", path)

	defer step.Cleanup(tempState)

	return pctx.uploadVolume(path)
}

func (vs *FilesVolumeSource) cdromStep(pctx *PreparationContext) multistep.Step {
	pctx.Ui.Message(fmt.Sprintf("Assembling CDROM %s/%s", pctx.VolumeConfig.Pool, pctx.VolumeConfig.Name))

	var files = []string{}
	var err error

	for _, path := range vs.Files {
		if strings.ContainsAny(path, "*?") {
			var globbedFiles []string
			globbedFiles, err = filepath.Glob(path)

			if len(globbedFiles) > 0 {
				files = append(files, globbedFiles...)
			}
		} else {
			_, err = os.Stat(path)
			files = append(files, path)
		}

		if err != nil {
			pctx.HaltOnError(err, "issues with file '%s': %s", path, err)
		}
	}

	createStep := commonsteps.StepCreateCD{
		Files:   files,
		Content: vs.Contents,
		Label:   vs.Label,
	}

	return &createStep
}

func (vs *FilesVolumeSource) floppyStep(pctx *PreparationContext) multistep.Step {
	pctx.Ui.Message(fmt.Sprintf("Assembling Floppy disk %s/%s", pctx.VolumeConfig.Pool, pctx.VolumeConfig.Name))

	createStep := commonsteps.StepCreateFloppy{
		Files:       []string{}, // StepCreateFloppy.Files doesn't preserve directory structure
		Directories: vs.Files,   // StepCreateFloppy.Directories preserves directory structure
		Content:     vs.Contents,
		Label:       vs.Label,
	}

	return &createStep
}

func (vs *FilesVolumeSource) path(state multistep.StateBag) (path string, err error) {
	var raw interface{}
	var ok bool
	switch vs.filesVolumeFormat {
	case FilesCdDeviceType:
		raw, ok = state.GetOk("cd_path")
	case FilesFloppyDeviceType:
		raw, ok = state.GetOk("floppy_path")
	default:
		err = fmt.Errorf("unknown format: %s", vs.filesVolumeFormat)
		return
	}

	if ok {
		path, ok = raw.(string)
	}

	if !ok || path == "" {
		err = fmt.Errorf("the interim state bag got corrupted")
	}

	return
}
