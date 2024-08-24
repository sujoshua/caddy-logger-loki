package caddy_logger_loki

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/loki/v3/clients/pkg/promtail/client"
	"github.com/prometheus/common/config"
	"io"
	"net/url"
	"strconv"
	"time"
)

func init() {
	caddy.RegisterModule(&LokiLog{})
}

/*
LokiLog is a Caddy logger used to send logs to Loki.
*/
type LokiLog struct {
	/*
	  The URL where Loki is listening, denoted in Loki as http_listen_address and
	  http_listen_port. If Loki is running in microservices mode, this is the HTTP
	  URL for the Distributor. Path to the push API needs to be included.
	  Example: http://example.com:3100/loki/api/v1/push
	*/
	Url string `json:"url,omitempty"`

	/*
	  Custom HTTP headers to be sent along with each push request.
	  Be aware that headers that are set by this writer itself (e.g. X-Scope-OrgID) can'T be overwritten.

	  Example: CF-Access-Client-Id: xxx
	  [ <labelname>: <labelvalue> ... ]
	*/
	Headers map[string]string `json:"headers,omitempty"`

	/*
	  The tenant ID used by default to push logs to Loki. If omitted or empty
	  it assumes Loki is running in single-tenant mode and no X-Scope-OrgID header
	  is sent.
	*/
	TenantId string `json:"tenant_id,omitempty"`

	/*
	  Maximum amount of time to wait before sending a batch, even if that
	  batch isn'T full.
	  default = 1s
	*/
	BatchWait StrTimeDuration `json:"batchwait,omitempty"`

	/*
	  Maximum batch size (in bytes) of logs to accumulate before sending
	  the batch to Loki.
	  default = 1048576
	*/
	BatchSize int `json:"batchsize,omitempty"`

	// If using basic auth, configures the username and password sent.
	BasicAuth *BasicAuth `json:"basic_auth,omitempty"`

	// Optional OAuth 2.0 configuration
	// Cannot be used at the same time as basic_auth or authorization
	Oauth2 *OAuth2 `json:"oauth2,omitempty"`

	// Bearer token to send to the server.
	BearerToken string `json:"bearer_token,omitempty"`

	// File containing bearer token to send to the server.
	BearTokenFile string `json:"bearer_token_file,omitempty"`

	// HTTP proxy server to use to connect to the server.
	ProxyURL string `json:"proxy_url,omitempty"`

	// If connecting to a TLS server, configures how the TLS authentication handshake will operate.
	TlsConfig TLSConfig `json:"tls_config,omitempty"`

	/*
	  Configures how to retry requests to Loki when a request
	  fails.
	  Default backoff schedule:
	  0.5s, 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s(4.267m)
	  For a total time of 511.5s(8.5m) before logs are lost
	*/
	BackoffConfig BackoffConfig `json:"backoff_config,omitempty"`

	/*
		Disable retries of batches that Loki responds to with a 429 status code (TooManyRequests). This reduces
		impacts on batches from other tenants, which could end up being delayed or dropped due to exponential backoff.
	*/
	DropRateLimitedBatches bool `json:"drop_rate_limited_batches,omitempty"`

	/*
		Static labels to add to all logs being sent to Loki.
		Use map like {"foo": "bar"} to add a label foo with
		value bar.
	*/
	Labels map[string]string `json:"labels,omitempty"`

	// Maximum time to wait for a server to respond to a request, default is 10s
	TimeOut StrTimeDuration `json:"timeout,omitempty"`

	// loki client config
	clientConfig client.Config

	/*
		Limits the max number of active streams.
		Limiting the number of streams is useful as a mechanism to limit memory usage by this instance, which helps
		to avoid OOM scenarios.
		0 means it is disabled.
		default is 0.
	*/
	MaxStreams int `json:"max_streams,omitempty"`

	// Maximum log line byte size allowed without dropping. Example: 256kb, 2M. 0 to disable. default is 0.
	MaxLineSize int `json:"max_line_size,omitempty"`

	// Whether to truncate lines that exceed max_line_size. No effect if max_line_size is disabled. default is false.
	MaxLineSizeTruncate bool `json:"max_line_size_truncate,omitempty"`

	// inner logger to log module itself log
	logger logger
}

type BackoffConfig struct {
	// Initial backoff time between retries, default is 500ms
	MinPeriod StrTimeDuration `json:"min_period,omitempty"`

	// Maximum backoff time between retries, default is 5m
	MaxPeriod StrTimeDuration `json:"max_period,omitempty"`

	// Maximum number of retries to do, default is 10
	MaxRetries int `json:"max_retries,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (l *LokiLog) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.logging.writers.loki",
		New: func() caddy.Module { return new(LokiLog) },
	}
}

// Provision sets up the module, now only init the logger.
func (l *LokiLog) Provision(ctx caddy.Context) error {
	l.logger = newLogger(ctx.Logger())
	return nil
}

/*
UnmarshalCaddyfile sets up the module from Caddyfile tokens. Syntax:

	url
	headers {
		key value
	}

	tenant_id
	batchwait
	batchsize

	basic_auth {
		username
		username_file
		password
		password_file
	}

	oauth2 {
		client_id
		client_secret
		scopes
		token_url
		endpoint_params {
			key value
		}
	}

	bearer_token
	bearer_token_file
	proxy_url
	tls_config {
		ca_file
		cert_file
		key_file
		server_name
		insecure_skip_verify
	}
	backoff_config {
		min_period
		max_period
		max_retries
	}
	drop_rate_limited_batches
	labels {
		key value
	}
	timeout
	max_streams
	max_line_size
	max_line_size_truncate
*/
func (l *LokiLog) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	if !d.NextArg() {
		return d.ArgErr()
	}
	for block := d.Nesting(); d.NextBlock(block); {
		switch d.Val() {
		case "url":
			if !d.NextArg() {
				return d.ArgErr()
			}
			l.Url = d.Val()
		case "headers":
			headers := map[string]string{}
			for nestingHeaders := d.Nesting(); d.NextBlock(nestingHeaders); {
				key := d.Val()

				if !d.NextArg() {
					return d.ArgErr()
				}

				headers[key] = d.Val()
			}
			l.Headers = headers
		case "tenant_id":
			if !d.NextArg() {
				return d.ArgErr()
			}
			l.TenantId = d.Val()

		case "batchwait":
			if !d.NextArg() {
				return d.ArgErr()
			}
			v := d.Val()
			err := l.BatchWait.FromString(v)
			if err != nil {
				return fmt.Errorf("parse batchwait parameter failed, invalid duration: %v", err)
			}
		case "batchsize":
			if !d.NextArg() {
				return d.ArgErr()
			}
			v := d.Val()
			s, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("parse batchsize parameter failed, invalid int: %v", err)
			}
			l.BatchSize = s
		case "basic_auth":
			l.BasicAuth = &BasicAuth{}
			for basicAuthBlock := d.Nesting(); d.NextBlock(basicAuthBlock); {
				switch d.Val() {
				case "username":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.BasicAuth.Username = d.Val()
				case "username_file":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.BasicAuth.UsernameFile = d.Val()
				case "password":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.BasicAuth.Password = Secret(d.Val())
				case "password_file":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.BasicAuth.PasswordFile = d.Val()
				}
			}
		case "oauth2":
			l.Oauth2 = &OAuth2{}
			for oauth2Block := d.Nesting(); d.NextBlock(oauth2Block); {
				switch d.Val() {
				case "client_id":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.Oauth2.ClientID = d.Val()
				case "client_secret":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.Oauth2.ClientSecret = Secret(d.Val())
				case "scopes":
					if !d.NextArg() {
						return d.ArgErr()
					}
					scopes := make([]string, 2)
					for d.NextArg() {
						scopes = append(scopes, d.Val())
					}
					l.Oauth2.Scopes = scopes
				case "token_url":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.Oauth2.TokenURL = d.Val()
				case "endpoint_params":
					endpointParams := map[string]string{}
					for oauth2EndpointParamsBlock := d.Nesting(); d.NextBlock(oauth2EndpointParamsBlock); {
						key := d.Val()

						if !d.NextArg() {
							return d.ArgErr()
						}

						endpointParams[key] = d.Val()
					}
					l.Oauth2.EndpointParams = endpointParams
				}
			}
		case "bearer_token":
			if !d.NextArg() {
				return d.ArgErr()
			}
			l.BearerToken = d.Val()
		case "bearer_token_file":
			if !d.NextArg() {
				return d.ArgErr()
			}
			l.BearTokenFile = d.Val()
		case "proxy_url":
			if !d.NextArg() {
				return d.ArgErr()
			}
			l.ProxyURL = d.Val()
		case "tls_config":
			for tlsConfigBlock := d.Nesting(); d.NextBlock(tlsConfigBlock); {
				switch d.Val() {
				case "ca_file":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.TlsConfig.CAFile = d.Val()
				case "cert_file":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.TlsConfig.CertFile = d.Val()
				case "key_file":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.TlsConfig.KeyFile = d.Val()
				case "server_name":
					if !d.NextArg() {
						return d.ArgErr()
					}
					l.TlsConfig.ServerName = d.Val()
				case "insecure_skip_verify":
					l.TlsConfig.InsecureSkipVerify = true
				}
			}
		case "backoff_config":
			for backoffConfigBlock := d.Nesting(); d.NextBlock(backoffConfigBlock); {
				switch d.Val() {
				case "min_period":
					if !d.NextArg() {
						return d.ArgErr()
					}
					v := d.Val()
					err := l.BackoffConfig.MinPeriod.FromString(v)
					if err != nil {
						return fmt.Errorf("parse min_period parameter failed, invalid duration: %v", err)
					}
				case "max_period":
					if !d.NextArg() {
						return d.ArgErr()
					}
					v := d.Val()
					err := l.BackoffConfig.MaxPeriod.FromString(v)
					if err != nil {
						return fmt.Errorf("parse max_period parameter failed, invalid duration: %v", err)
					}
				case "max_retries":
					if !d.NextArg() {
						return d.ArgErr()
					}
					v := d.Val()
					i, err := strconv.Atoi(v)
					if err != nil {
						return fmt.Errorf("parse max_retries parameter failed, invalid int: %v", err)
					}
					l.BackoffConfig.MaxRetries = i
				}
			}
		case "drop_rate_limited_batches":
			l.DropRateLimitedBatches = true
		case "labels":
			labels := map[string]string{}
			for nestingLabels := d.Nesting(); d.NextBlock(nestingLabels); {
				key := d.Val()

				if !d.NextArg() {
					return d.ArgErr()
				}

				labels[key] = d.Val()
			}
			l.Labels = labels
		case "max_streams":
			if !d.NextArg() {
				return d.ArgErr()
			}
			v := d.Val()
			i, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("parse max_streams parameter failed, invalid int: %v", err)
			}
			l.MaxStreams = i
		case "max_line_size":
			if !d.NextArg() {
				return d.ArgErr()
			}
			v := d.Val()
			i, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("parse max_line_size parameter failed, invalid int: %v", err)
			}
			l.MaxLineSize = i
		case "max_line_size_truncate":
			l.MaxLineSizeTruncate = true
		case "timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			v := d.Val()
			err := l.TimeOut.FromString(v)
			if err != nil {
				return fmt.Errorf("parse timeout parameter failed, invalid duration: %v", err)
			}
		}
	}
	return nil
}

// Validate ensures the module is properly configured.
func (l *LokiLog) Validate() error {
	name := "caddy-logger-loki"

	if l.Url == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.Parse(l.Url)
	if err != nil {
		return fmt.Errorf("url is invalid: %v", err)
	}
	u2 := flagext.URLValue{URL: u}

	if len(l.Labels) == 0 {
		return fmt.Errorf("labels is nil, at least one label is required")
	}

	if l.BatchWait.T == 0 {
		l.BatchWait.T = 1 * time.Second
	}

	if l.BatchSize == 0 {
		l.BatchSize = 1048576
	}

	if l.TimeOut.T == 0 {
		l.TimeOut.T = 10 * time.Second
	}

	var proxyURL *url.URL
	if l.ProxyURL != "" {
		proxyURL, err = url.Parse(l.ProxyURL)
		if err != nil {
			return fmt.Errorf("proxy_url is invalid: %v", err)
		}
	}

	if l.BackoffConfig.MaxRetries == 0 {
		l.BackoffConfig.MaxRetries = 10
	}
	if l.BackoffConfig.MinPeriod.T == 0 {
		l.BackoffConfig.MinPeriod.T = 500 * time.Millisecond
	}
	if l.BackoffConfig.MaxPeriod.T == 0 {
		l.BackoffConfig.MaxPeriod.T = 5 * time.Minute
	}
	backoffConfig := backoff.Config{
		MinBackoff: l.BackoffConfig.MinPeriod.TimeDuration(),
		MaxBackoff: l.BackoffConfig.MaxPeriod.TimeDuration(),
		MaxRetries: l.BackoffConfig.MaxRetries,
	}

	var basicAuth *config.BasicAuth
	if l.BasicAuth != nil {
		basicAuth = l.BasicAuth.ToPrometheusBasicAuth()
	}
	var oauth2 *config.OAuth2
	if l.Oauth2 != nil {
		oauth2 = l.Oauth2.ToPrometheusOAuth2()
	}

	l.clientConfig = client.Config{
		Name:      name,
		URL:       u2,
		BatchWait: l.BatchWait.TimeDuration(),
		BatchSize: l.BatchSize,
		Client: config.HTTPClientConfig{
			BasicAuth:       basicAuth,
			OAuth2:          oauth2,
			BearerToken:     config.Secret(l.BearerToken),
			BearerTokenFile: l.BearTokenFile,
			TLSConfig:       l.TlsConfig.ToPrometheusTLSConfig(),
			ProxyConfig: config.ProxyConfig{
				ProxyURL: config.URL{URL: proxyURL},
			},
		},
		Headers:                l.Headers,
		BackoffConfig:          backoffConfig,
		Timeout:                l.TimeOut.TimeDuration(),
		TenantID:               l.TenantId,
		DropRateLimitedBatches: l.DropRateLimitedBatches,
	}

	return nil
}

func (l *LokiLog) String() string {
	return "loki"
}

func (l *LokiLog) WriterKey() string {
	return fmt.Sprintf("loki_log_%s", l.Url)
}

func (l *LokiLog) OpenWriter() (io.WriteCloser, error) {
	// TODO: add metrics support.
	metric := client.NewMetrics(nil)
	c, err := client.New(metric, l.clientConfig, l.MaxStreams, l.MaxLineSize, l.MaxLineSizeTruncate, l.logger)
	if err != nil {
		return nil, err
	}

	writer := newLokiWriter(c, l.logger, l.Labels)

	return writer, nil
}

// Interface guards
var (
	_ caddy.Provisioner     = (*LokiLog)(nil)
	_ caddy.WriterOpener    = (*LokiLog)(nil)
	_ caddyfile.Unmarshaler = (*LokiLog)(nil)
)
