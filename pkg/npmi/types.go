package npmi

import "strings"

type LogLevel int32

const (
	NoLevel LogLevel = iota
	Info
	Debug
	Trace
)

func LogLevelFromString(levelStr string) LogLevel {
	levelStr = strings.ToLower(levelStr)
	switch levelStr {
	case "trace":
		return Trace
	case "debug":
		return Debug
	case "info":
		return Info
	default:
		return NoLevel
	}
}

func (l LogLevel) String() string {
	switch l {
	case Trace:
		return "trace"
	case Debug:
		return "debug"
	case Info:
		return "info"
	case NoLevel:
		return "none"
	default:
		return "unknown"
	}
}

// MinioCacheOptions contains configuration for Minio Cache
type MinioCacheOptions struct {
	Endpoint        string `env:"NPMI_MINIO_ENDPOINT"`
	AccessKeyID     string `env:"NPMI_MINIO_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"NPMI_MINIO_SECRET_ACCESS_KEY"`
	Bucket          string `env:"NPMI_MINIO_BUCKET"`
	UseTLS          bool   `env:"NPMI_MINIO_TLS"`
	InsecureTLS     bool   `env:"NPMI_MINIO_TLS_INSECURE"`
}

// LocalCacheOptions constains configuration for Local Cache
type LocalCacheOptions struct {
	Dir string `env:"NPMI_LOCAL_DIR"`
}

// Options describes the runtime configuration
type Options struct {
	Force           bool `env:"NPMI_FORCE"`
	LocalCache      *LocalCacheOptions
	LogLevel        LogLevel `env:"NPMI_LOGLEVEL"`
	MinioCache      *MinioCacheOptions
	PrecacheCommand string `env:"NPMI_PRECACHE"`
	TempDir         string `env:"NPMI_TEMP_DIR"`
	UseLocalCache   bool   `env:"NPMI_LOCAL"`
	UseMinioCache   bool   `env:"NPMI_MINIO"`
}
