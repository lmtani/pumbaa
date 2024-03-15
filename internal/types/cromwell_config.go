package types

type BackendConfig struct {
	Default   string
	Providers []ProviderConfig
}

type ProviderConfig struct {
	Name        string
	ActorFactor string
	Config      ProviderSettings
}

type ProviderSettings struct {
	MaxConcurrentWorkflows int
	ConcurrentJobLimit     int
	FileSystems            Engine
}

type GcsFilesystem struct {
	Auth    string
	Enabled bool
}

type LocalFilesystem struct {
	Localization []string
}

type Filesystems struct {
	GcsFilesystem   GcsFilesystem
	HTTP            struct{}
	LocalFilesystem LocalFilesystem
}

type Engine struct {
	Filesystems
}

type Database struct {
	Profile           string
	Driver            string
	URL               string
	Host              string
	Port              int
	User              string
	Password          string
	ConnectionTimeout int
}

type CallCaching struct {
	Enabled                   bool
	InvalidateBadCacheResults bool
}

type Docker struct {
	PerformRegistryLookupIfDigestIsProvided bool
}

type Config struct {
	Override      bool
	BackendConfig BackendConfig
	Database      Database
	CallCaching   CallCaching
	Docker        Docker
	Engine        Engine
}
