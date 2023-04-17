package cromwell

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
	Auth string `json:"auth"`
}

type LocalFilesystem struct {
	Localization []string `json:"localization"`
}

type Filesystems struct {
	GcsFilesystem   `json:"gcs,omitempty"`
	HTTP            struct{} `json:"http,omitempty"`
	LocalFilesystem `json:"local,omitempty"`
}

type Engine struct {
	Filesystems `json:"filesystems"`
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
	BackendConfig
	Database
	CallCaching
	Docker
	Engine
}
