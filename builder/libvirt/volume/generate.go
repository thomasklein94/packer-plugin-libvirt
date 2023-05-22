package volume

//go:generate go run github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc mapstructure-to-hcl2 -type Volume,VolumeSource,ExternalVolumeSource,FilesVolumeSource,CloudInitSource,BackingStoreVolumeSource,CloningVolumeSource
