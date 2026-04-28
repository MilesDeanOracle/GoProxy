package config

// Config contains the proxy server runtime configuration.
type Config struct {
	Server ServerConfig `yaml:"server" json:"server"`
	Relay  RelayConfig  `yaml:"relay" json:"relay"`
	Log    LogConfig    `yaml:"log" json:"log"`
	UI     UIConfig     `yaml:"ui" json:"ui"`
}

// ServerConfig contains inbound protocol listener settings.
type ServerConfig struct {
	SOCKS5 ProtocolConfig `yaml:"socks5" json:"socks5"`
	HTTP   ProtocolConfig `yaml:"http" json:"http"`
}

// ProtocolConfig contains one inbound protocol listener setting.
type ProtocolConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Host    string `yaml:"host" json:"host"`
	Port    int    `yaml:"port" json:"port"`
}

// RelayConfig contains connection relay limits and timeouts.
type RelayConfig struct {
	DialTimeoutSec int `yaml:"dial_timeout_sec" json:"dialTimeoutSec"`
	ReadTimeoutSec int `yaml:"read_timeout_sec" json:"readTimeoutSec"`
	MaxConnections int `yaml:"max_connections" json:"maxConnections"`
	KeepAliveSec   int `yaml:"keepalive_sec" json:"keepaliveSec"`
}

// LogConfig contains structured logging and rotation settings.
type LogConfig struct {
	Level      string `yaml:"level" json:"level"`
	MaxSizeMB  int    `yaml:"max_size_mb" json:"maxSizeMb"`
	MaxBackups int    `yaml:"max_backups" json:"maxBackups"`
	Output     string `yaml:"output" json:"output"`
}

// UIConfig contains desktop UI preferences.
type UIConfig struct {
	Theme          string `yaml:"theme" json:"theme"`
	Language       string `yaml:"language" json:"language"`
	StartMinimized bool   `yaml:"start_minimized" json:"startMinimized"`
	ShowTrayIcon   bool   `yaml:"show_tray_icon" json:"showTrayIcon"`
}

// Default returns a validated baseline configuration.
func Default() Config {
	return Config{
		Server: ServerConfig{
			SOCKS5: ProtocolConfig{
				Enabled: true,
				Host:    "0.0.0.0",
				Port:    1080,
			},
			HTTP: ProtocolConfig{
				Enabled: true,
				Host:    "0.0.0.0",
				Port:    8080,
			},
		},
		Relay: RelayConfig{
			DialTimeoutSec: 10,
			ReadTimeoutSec: 30,
			MaxConnections: 1000,
			KeepAliveSec:   15,
		},
		Log: LogConfig{
			Level:      "info",
			MaxSizeMB:  50,
			MaxBackups: 3,
			Output:     "both",
		},
		UI: UIConfig{
			Theme:          "auto",
			Language:       "zh-CN",
			StartMinimized: false,
			ShowTrayIcon:   true,
		},
	}
}
