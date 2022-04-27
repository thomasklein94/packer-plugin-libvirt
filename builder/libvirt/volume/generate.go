package volume

//go:generate packer-sdc mapstructure-to-hcl2 -type Volume,VolumeSource,HttpVolumeSource,CloudInitSource,BackingStoreVolumeSource
