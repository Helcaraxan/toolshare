package config

type StorageGitHub struct {
	Slug                 string `mapstructure:"slug"`
	ReleaseAssetTemplate string `mapstructure:"assetTemplate"`
	ArchivePath          string `mapstructure:"archivePathTemplate"`
}

type StorageGCS struct{}

type StorageLocal struct{}

type StorageS3 struct{}

func (s *StorageLocal) isStorage()  {}
func (s *StorageGitHub) isStorage() {}
func (s *StorageGCS) isStorage()    {}
func (s *StorageS3) isStorage()     {}
