package certainly

// DNSConfig holds the config structure
type CertainlyCFG struct {
	General         general
	NS              nameserver
	Rewrites        map[string]string `toml:"rewrites"`
	Logconfig       logconfig
	Notification    notifications
	HTTPD           httpd
	HTTPDInjections map[string]string `toml:"httpd_injection_templates"`
}

type httpd struct {
	HTTPPort                  string   `toml:"http_port"`
	HTTPSPort                 string   `toml:"https_port"`
	InjectionTemplateFilepath string   `toml:"injection_template_filepath"`
	InjectionFilters          []string `toml:"injection_filters"`
}

type general struct {
	IP               string   `toml:"ip"`
	Debug            bool     `toml:"debug"`
	ACMECacheDir     string   `toml:"cert_dir"`
	TLSFilters       []string `toml:"tls_filters"`
	TLSUpstreamCheck bool     `toml:"tls_upstream_check"`
}

// Config file nameserver section
type nameserver struct {
	Port          string   `toml:"port"`
	Proto         string   `toml:"protocol"`
	DefaultDomain string   `toml:"default_domain"`
	Domains       []string `toml:"domains"`
	Nsname        string   `toml:"nsname"`
	Nsadmin       string   `toml:"nsadmin"`
	NSResponseIP  string   `toml:"ns_response_ip"`
	Debug         bool     `toml:"debug"`
	StaticRecords []string `toml:"records"`
	Ttl           int      `toml:"ttl"`
}

// Logging config
type logconfig struct {
	Level   string `toml:"loglevel"`
	Logtype string `toml:"logtype"`
	File    string `toml:"logfile"`
	Format  string `toml:"logformat"`
}

// Notification config
type notifications struct {
	Slack               bool     `toml:"slack"`
	SlackToken          string   `toml:"slack_token"`
	SlackDefaultChannel string   `toml:"slack_default_channel"`
	SlackHTTPChannel    string   `toml:"slack_http_channel"`
	SlackDNSChannel     string   `toml:"slack_dns_channel"`
	SlackSMTPChannel    string   `toml:"slack_smtp_channel"`
	SlackIMAPChannel    string   `toml:"slack_imap_channel"`
	HTTP                bool     `toml:"http"`
	HTTPFilters         []string `toml:"http_filters"`
	DNS                 bool     `toml:"dns"`
	DNSFilters          []string `toml:"dns_filters"`
	SMTP                bool     `toml:"smtp"`
	SMTPFilters         []string `toml:"smtp_filters"`
	IMAP                bool     `toml:"imap"`
	IMAPFilters         []string `toml:"imap_filters"`
}

// UpstreamNSRecord is used for target nameserver records
type UpstreamNSRecord struct {
	Addr string
	IPv6 bool
}
