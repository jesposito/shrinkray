package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// MediaPath is the root directory to browse for media files
	MediaPath string `yaml:"media_path"`

	// TempPath is where temp files are written during transcoding
	// If empty, temp files go in the same directory as the source
	TempPath string `yaml:"temp_path"`

	// OriginalHandling determines what happens to original files after transcoding
	// Options: "replace" (rename original to .old), "keep" (keep original, new file replaces)
	OriginalHandling string `yaml:"original_handling"`

	// Workers is the number of concurrent transcode jobs (default 1)
	Workers int `yaml:"workers"`

	// FFmpegPath is the path to ffmpeg binary (default: "ffmpeg")
	FFmpegPath string `yaml:"ffmpeg_path"`

	// FFprobePath is the path to ffprobe binary (default: "ffprobe")
	FFprobePath string `yaml:"ffprobe_path"`

	// QueueFile is where the job queue is persisted (default: config dir + queue.json)
	QueueFile string `yaml:"queue_file"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MediaPath:        "/media",
		TempPath:         "", // same directory as source
		OriginalHandling: "replace",
		Workers:          1,
		FFmpegPath:       "ffmpeg",
		FFprobePath:      "ffprobe",
		QueueFile:        "",
	}
}

// Load reads config from a YAML file, applying defaults for missing values
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file - use defaults
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for empty values
	if cfg.FFmpegPath == "" {
		cfg.FFmpegPath = "ffmpeg"
	}
	if cfg.FFprobePath == "" {
		cfg.FFprobePath = "ffprobe"
	}
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}

	return cfg, nil
}

// Save writes the config to a YAML file
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetTempDir returns the directory for temp files
// If TempPath is set, returns that; otherwise returns the directory of the source file
func (c *Config) GetTempDir(sourcePath string) string {
	if c.TempPath != "" {
		return c.TempPath
	}
	return filepath.Dir(sourcePath)
}
