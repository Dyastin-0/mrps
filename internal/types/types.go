package types

import (
	"net"
	"net/http"
	"time"

	"github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"golang.org/x/time/rate"
)

type DomainsConfig map[string]Config

type ClientLimiter struct {
	Limiter  *rate.Limiter
	LastReq  time.Time
	Cooldown time.Time
}

const (
	HTTPProtocol = "http"
	TCPProtocol  = "tcp"
)

type Config struct {
	Enabled      bool            `yaml:"enabled"`
	Routes       RouteConfig     `yaml:"routes,omitempty"`
	SortedRoutes []string        `yaml:"-"`
	RateLimit    RateLimitConfig `yaml:"rate_limit,omitempty"`
	Protocol     string          `yaml:"protocol,omitempty"`
	EnableHTTP   bool            `yaml:"enable_http,omitempty"`
}

type RouteConfig map[string]PathConfig

type PathConfig struct {
	Dests        []Dest               `json:"Dests,omitempty" yaml:"dests,omitempty"`
	RewriteRule  rewriter.RewriteRule `yaml:"rewrite,omitempty"`
	BalancerType string               `yaml:"balancer,omitempty"`
	Balancer     Balancer             `yaml:"-"`
	BalancerTCP  BalancerTCP          `yaml:"-"`
}

type Dest struct {
	URL        string `yaml:"url"`
	WithTLS    bool   `yaml:"with_tls"`
	ServerName string `yaml:"server_name,omitempty"`
	Weight     int    `yaml:"weight,omitempty"`
}

type RateLimitConfig struct {
	Burst           int           `yaml:"burst,omitempty"`
	Rate            rate.Limit    `yaml:"rate,omitempty"`
	Cooldown        int64         `yaml:"cooldown,omitempty"`
	DefaultCooldown time.Duration `yaml:"-"`
}

type MiscConfig struct {
	Email               string   `yaml:"email,omitempty"`
	Secure              bool     `yaml:"secure"`
	MetricsEnabled      bool     `yaml:"enable_metrics"`
	MetricsPort         string   `yaml:"metrics_port,omitempty"`
	APIEnabled          bool     `yaml:"enable_api,omitempty"`
	ConfigAPIPort       string   `yaml:"api_port,omitempty"`
	AllowedOrigins      []string `yaml:"allowed_origins,omitempty"`
	Domain              string   `yaml:"domain,omitempty"`
	IP                  string   `yaml:"ip,omitempty"`
	AllowHTTP           bool     `yaml:"allow_http"`
	HealthCheckInterval int64    `yaml:"health_check_interval,omitempty"`
}

type YAML struct {
	Domains   DomainsConfig   `yaml:"domains,omitempty"`
	Misc      MiscConfig      `yaml:"misc,omitempty"`
	RateLimit RateLimitConfig `yaml:"rate_limit,omitempty"`
}

type Balancer interface {
	Serve(w http.ResponseWriter, r *http.Request, retries int) bool
	First() *common.Dest
	GetDests() []*common.Dest
	StopHealthChecks()
}

type BalancerTCP interface {
	Serve(conn net.Conn, sni string) bool
	First() *common.Dest
	GetDests() []*common.Dest
	StopHealthChecks()
}
