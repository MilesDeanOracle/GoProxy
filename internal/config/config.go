package config

// Config contains the proxy server runtime configuration.
type Config struct {
	Server ServerConfig `yaml:"server" json:"server"`
	Auth   AuthConfig   `yaml:"auth" json:"auth"`
	Relay  RelayConfig  `yaml:"relay" json:"relay"`
	Log    LogConfig    `yaml:"log" json:"log"`
	UI     UIConfig     `yaml:"ui" json:"ui"`
	Route  RouteConfig  `yaml:"route" json:"route"`
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

// AuthConfig contains optional username/password authentication settings.
type AuthConfig struct {
	Enabled bool       `yaml:"enabled" json:"enabled"`
	Users   []AuthUser `yaml:"users" json:"users"`
}

// AuthUser stores a username and bcrypt password hash.
type AuthUser struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
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
	Theme           string `yaml:"theme" json:"theme"`
	Language        string `yaml:"language" json:"language"`
	StartMinimized  bool   `yaml:"start_minimized" json:"startMinimized"`
	AutoStartProxy  bool   `yaml:"auto_start_proxy" json:"autoStartProxy"`
	ShowTrayIcon    bool   `yaml:"show_tray_icon" json:"showTrayIcon"`
	CloseToTray     bool   `yaml:"close_to_tray" json:"closeToTray"`
	TrayStatusAndIP bool   `yaml:"tray_status_and_ip" json:"trayStatusAndIp"`
}

// RouteConfig contains route policy runtime switches.
type RouteConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	ActiveFile string `yaml:"active_file" json:"activeFile"`
}

// RouteRuleSet stores a complete .rule file.
type RouteRuleSet struct {
	Name        string      `yaml:"name" json:"name"`
	Version     int         `yaml:"version" json:"version"`
	UpdatedAt   string      `yaml:"updated_at" json:"updatedAt"`
	Description string      `yaml:"description" json:"description"`
	Rules       []RouteRule `yaml:"rules" json:"rules"`
}

// RouteRule describes one destination matching and outbound binding rule.
type RouteRule struct {
	ID        string          `yaml:"id" json:"id"`
	Name      string          `yaml:"name" json:"name"`
	Enabled   bool            `yaml:"enabled" json:"enabled"`
	Priority  int             `yaml:"priority" json:"priority"`
	Protocols []string        `yaml:"protocols" json:"protocols"`
	MatchType string          `yaml:"match_type" json:"matchType"`
	Targets   []string        `yaml:"targets" json:"targets"`
	Outbound  OutboundBinding `yaml:"outbound" json:"outbound"`
	Remark    string          `yaml:"remark" json:"remark"`
}

// OutboundBinding chooses how a matched connection binds its local address.
type OutboundBinding struct {
	Mode      string `yaml:"mode" json:"mode"`
	LocalIP   string `yaml:"local_ip" json:"localIp"`
	Interface string `yaml:"interface" json:"interface"`
}

// RouteFileInfo describes one route policy file for UI selection.
type RouteFileInfo struct {
	Name      string `json:"name"`
	IsActive  bool   `json:"isActive"`
	UpdatedAt string `json:"updatedAt"`
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
		Auth: AuthConfig{
			Enabled: false,
			Users:   []AuthUser{},
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
			Theme:           "auto",
			Language:        "zh-CN",
			StartMinimized:  false,
			AutoStartProxy:  true,
			ShowTrayIcon:    true,
			CloseToTray:     true,
			TrayStatusAndIP: true,
		},
		Route: RouteConfig{
			Enabled:    false,
			ActiveFile: "default.rule",
		},
	}
}
