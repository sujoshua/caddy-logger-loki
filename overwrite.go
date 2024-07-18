package caddy_logger_loki

import "github.com/prometheus/common/config"

/*
	The Config struct used to new Client contains the is github.com/prometheus/common/config.Secret type
and this type has a special implementation for json.Marshaler interfaces. As a result, all Secret type fields
are marshal to "<secret>" string whatever plain secret is.
	However, when Caddyfile is used, caddy will firstly call UnmarshalCaddyfile method to fill LokiLog struct
and then marshal it to json format and then use this json as the configuration for the logger.
	So if we directly use the Secret type in the LokiLog struct, the Secret type will be marshal to "<secret>",
we can't get the real secret value from the configuration file.
	To solve this problem, we need to overwrite the Secret type in the LokiLog struct with a string type and
implement the UnmarshalCaddyfile and MarshalJSON methods to convert the string to Secret type and vice versa.
	This is what this file does.
*/

type Secret string

type BasicAuth struct {
	config.BasicAuth `json:",inline"`
	Password         Secret `json:"password,omitempty"`
}

// ToPrometheusBasicAuth converts BasicAuth to config.BasicAuth.
func (b BasicAuth) ToPrometheusBasicAuth() *config.BasicAuth {
	b.BasicAuth.Password = config.Secret(b.Password)
	return &b.BasicAuth
}

type TLSConfig struct {
	config.TLSConfig `json:",inline"`
	Key              Secret `json:"key,omitempty"`
}

// ToPrometheusTLSConfig converts TLSConfig to config.TLSConfig.
func (t TLSConfig) ToPrometheusTLSConfig() config.TLSConfig {
	t.TLSConfig.Key = config.Secret(t.Key)
	return t.TLSConfig
}

type OAuth2 struct {
	config.OAuth2 `json:",inline"`
	TlsConfig     TLSConfig `json:"tls_config,omitempty"`
	ClientSecret  Secret    `json:"client_secret,omitempty"`
}

// ToPrometheusOAuth2 converts OAuth2 to config.OAuth2.
func (o OAuth2) ToPrometheusOAuth2() *config.OAuth2 {
	o.OAuth2.ClientSecret = config.Secret(o.ClientSecret)
	o.OAuth2.TLSConfig = o.TlsConfig.ToPrometheusTLSConfig()
	return &o.OAuth2
}
