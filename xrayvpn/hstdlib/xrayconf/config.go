package xrayconf

type Config struct {
	Remarks   string         `json:"remarks,omitempty"`
	Log       LogConfig      `json:"log"`
	DNS       *DNSConfig     `json:"dns,omitempty"`
	Inbounds  []Inbound      `json:"inbounds"`
	Outbounds []Outbound     `json:"outbounds"`
	Routing   *RoutingConfig `json:"routing,omitempty"`
}

type LogConfig struct {
	LogLevel string `json:"loglevel"`
	DNSLog   bool   `json:"dnsLog,omitempty"`
}

type DNSConfig struct {
	Servers []string `json:"servers"`
}

type Inbound struct {
	Tag            string        `json:"tag,omitempty"`
	Listen         string        `json:"listen"`
	Port           uint16        `json:"port"`
	Protocol       string        `json:"protocol"`
	Settings       any           `json:"settings,omitempty"`
	StreamSettings *StreamConfig `json:"streamSettings,omitempty"`
	Sniffing       *Sniffing     `json:"sniffing,omitempty"`
}

type Sniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride,omitempty"`
}

type VLESSInboundSettings struct {
	Clients    []VLESSAccount `json:"clients"`
	Decryption string         `json:"decryption"`
}

type VLESSAccount struct {
	ID         string `json:"id"`
	Flow       string `json:"flow,omitempty"`
	Encryption string `json:"encryption,omitempty"`
}

type Outbound struct {
	Tag            string        `json:"tag"`
	Protocol       string        `json:"protocol"`
	Settings       any           `json:"settings,omitempty"`
	StreamSettings *StreamConfig `json:"streamSettings,omitempty"`
}

type VLESSOutboundSettings struct {
	Vnext []VLESSServer `json:"vnext"`
}

type VLESSServer struct {
	Address string         `json:"address"`
	Port    uint16         `json:"port"`
	Users   []VLESSAccount `json:"users"`
}

type FreedomSettings struct {
	DomainStrategy string `json:"domainStrategy,omitempty"`
}

type SocksSettings struct {
	UDP bool `json:"udp"`
}

type StreamConfig struct {
	Network         string         `json:"network"`
	Security        string         `json:"security"`
	TLSSettings     *TLSConfig     `json:"tlsSettings,omitempty"`
	REALITYSettings *RealityConfig `json:"realitySettings,omitempty"`
	TCPSettings     *TCPConfig     `json:"tcpSettings,omitempty"`
	KCPSettings     *KCPConfig     `json:"kcpSettings,omitempty"`
	WSSettings      *WSConfig      `json:"wsSettings,omitempty"`
	GRPCSettings    *GRPCConfig    `json:"grpcSettings,omitempty"`
	XHTTPSettings   *XHTTPConfig   `json:"xhttpSettings,omitempty"`
	SocketSettings  *SocketConfig  `json:"sockopt,omitempty"`
}

type TLSConfig struct {
	ServerName   string        `json:"serverName,omitempty"`
	Certificates []Certificate `json:"certificates,omitempty"`
}

type Certificate struct {
	CertificateFile string `json:"certificateFile"`
	KeyFile         string `json:"keyFile"`
}

type RealityConfig struct {
	Fingerprint string   `json:"fingerprint,omitempty"`
	ServerName  string   `json:"serverName,omitempty"`
	Target      string   `json:"target,omitempty"`
	ServerNames []string `json:"serverNames,omitempty"`
	PublicKey   string   `json:"publicKey,omitempty"`
	PrivateKey  string   `json:"privateKey,omitempty"`
	ShortID     string   `json:"shortId,omitempty"`
	ShortIDs    []string `json:"shortIds,omitempty"`
}

type TCPConfig struct {
	Header any `json:"header,omitempty"`
}

type WSConfig struct {
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
}

type GRPCConfig struct {
	ServiceName string `json:"serviceName"`
	MultiMode   bool   `json:"multiMode,omitempty"`
}

type KCPConfig struct {
	Seed   string `json:"seed,omitempty"`
	Header any    `json:"header,omitempty"`
}

type XHTTPConfig struct {
	Path              string `json:"path"`
	Mode              string `json:"mode,omitempty"`
	XPaddingBytes     string `json:"xPaddingBytes,omitempty"`
	XPaddingObfsMode  bool   `json:"xPaddingObfsMode,omitempty"`
	XPaddingPlacement string `json:"xPaddingPlacement,omitempty"`
	XPaddingKey       string `json:"xPaddingKey,omitempty"`
	UplinkHTTPMethod  string `json:"uplinkHTTPMethod,omitempty"`
	NoGRPCHeader      bool   `json:"noGRPCHeader,omitempty"`
	NoSSEHeader       bool   `json:"noSSEHeader,omitempty"`
}

type SocketConfig struct {
	Mark int32 `json:"mark,omitempty"`
}

type RoutingConfig struct {
	DomainStrategy string      `json:"domainStrategy"`
	Rules          []RouteRule `json:"rules"`
}

type RouteRule struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	InboundTag  string   `json:"inboundTag,omitempty"`
	IP          []string `json:"ip,omitempty"`
	Domain      []string `json:"domain,omitempty"`
	Protocol    []string `json:"protocol,omitempty"`
	Network     string   `json:"network,omitempty"`
	Port        string   `json:"port,omitempty"`
}
