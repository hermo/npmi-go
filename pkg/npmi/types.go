package npmi

// MinioCacheOptions contains configuration for Minio Cache
type MinioCacheOptions struct {
	Endpoint        string `env:"NPMI_MINIO_ENDPOINT"`
	AccessKeyID     string `env:"NPMI_MINIO_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"NPMI_MINIO_SECRET_ACCESS_KEY"`
	Bucket          string `env:"NPMI_MINIO_BUCKET"`
	UseTLS          bool   `env:"NPMI_MINIO_TLS"`
}

// LocalCacheOptions constains configuration for Local Cache
type LocalCacheOptions struct {
	Dir string `env:"NPMI_LOCAL_DIR"`
}

// Options describes the runtime configuration
type Options struct {
	Verbose         bool `env:"NPMI_VERBOSE"`
	Force           bool `env:"NPMI_FORCE"`
	UseLocalCache   bool `env:"NPMI_LOCAL"`
	UseMinioCache   bool `env:"NPMI_MINIO"`
	MinioCache      *MinioCacheOptions
	LocalCache      *LocalCacheOptions
	PrecacheCommand string `env:"NPMI_PRECACHE"`
}
