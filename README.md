# Caddy Logger Loki Plugin

A plugin of caddy to push log directly to loki without promtail and parameters are **compliable** with promtail client configuration section.

## Usage
### parameters:
Most of the parameters are same as promtail client, so you can check [promtail client doc](https://grafana.com/docs/loki/latest/send-data/promtail/configuration/#clients) as well.

different parameters are:

| parameter | type | description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                | default |
|:---------:|:----:|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-------:|
| `labels`  | map  | Static labels to add to all logs being sent to Loki.  Use map like {"foo": "bar"} to add a label foo with value bar. Support caddy all [placeholders](https://caddyserver.com/docs/conventions#placeholders) except http related. Unlike Promtail, you **MUST** set at least one label, because plugin won't add any.  It's actually is `external_labels` filed in promtail, but we can't set labels in cmd, it's the only way to add labels, so there shouldn't have concept of external. |    -    |

same parameters are:

|             parameter             |  type  | description                                                                                                                                                                                                                                                                    | default |
|:---------------------------------:|:------:|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-------:|
|               `url`               | string | The URL where Loki is listening, denoted in Loki as http_listen_address and http_listen_port. If Loki is running in microservices mode, this is the HTTP URL for the Distributor. Path to the push API needs to be included. Example: http://example.com:3100/loki/api/v1/push |    -    |
|             `headers`             |  map   | Custom HTTP headers to be sent along with each push request. Be aware that headers that are set by Promtail itself (e.g. X-Scope-OrgID) can't be overwritten.                                                                                                                  |    -    |
|            `tenant_id`            | string | The tenant ID used by default to push logs to Loki. If omitted or empty it assumes Loki is running in single-tenant mode and no X-Scope-OrgID header is sent.                                                                                                                  |    -    |
|            `batchwait`            | string | Maximum amount of time to wait before sending a batch, even if that batch isn't full.                                                                                                                                                                                          |   1s    |
|            `batchsize`            |  int   | Maximum batch size (in bytes) of logs to accumulate before sending the batch to Loki.                                                                                                                                                                                          | 1048576 |
|           `basic_auth`            |  map   | If using basic auth, configures the username and password sent.                                                                                                                                                                                                                |    -    |
|       `basic_auth.username`       | string | The username to use for basic auth.                                                                                                                                                                                                                                            |    -    |
|       `basic_auth.password`       | string | The password to use for basic auth.                                                                                                                                                                                                                                            |    -    |
|    `basic_auth.password_file`     | string | The file containing the password for basic auth.                                                                                                                                                                                                                               |    -    |
|             `oauth2`              |  map   | Optional OAuth 2.0 configuration. Cannot be used at the same time as basic_auth or authorization                                                                                                                                                                               |    -    |
|        `oauth2.client_id`         | string | Client id for oatuh2                                                                                                                                                                                                                                                           |    -    |
|      `oauth2.client_secret`       | string | Client secret for oatuh2                                                                                                                                                                                                                                                       |    -    |
|    `oauth2.clienn_secret_file`    | string | Read the client secret from a file. It is mutually exclusive with `oauth2.client_secret`                                                                                                                                                                                       |    -    |
|          `oauth2.scopes`          | string | Optional scopes for the token request.                                                                                                                                                                                                                                         |    -    |
|        `oauth2.token_url`         | string | The URL to fetch the token from.                                                                                                                                                                                                                                               |    -    |
|     `oauth2.endpoint_params`      |  map   | Optional parameters to append to the token URL                                                                                                                                                                                                                                 |    -    |
|          `bearer_token `          | string | Bearer token to send to the server.                                                                                                                                                                                                                                            |    -    |
|        `bearer_token_file`        | string | File containing bearer token to send to the server.                                                                                                                                                                                                                            |    -    |
|            `proxy_url`            | string | HTTP proxy server to use to connect to the server.                                                                                                                                                                                                                             |    -    |
|           `tls_config`            |  map   | If connecting to a TLS server, configures how the TLS authentication handshake will operate.                                                                                                                                                                                   |    -    |
|       `tls_config.ca_file`        | string | The CA file to use to verify the server.                                                                                                                                                                                                                                       |    -    |
|      `tls_config.cert_file`       | string | The cert file to send to the server for client auth.                                                                                                                                                                                                                           |    -    |
|       `tls_config.key_file`       | string | The key file to send to the server for client auth.                                                                                                                                                                                                                            |    -    |
|     `tls_config.server_name`      | string | TValidates that the server name in the server's certificate is this value.                                                                                                                                                                                                     |    -    |
| `tls_config.insecure_skip_verify` | string | If true, ignores the server certificate being signed by an unknown CA.                                                                                                                                                                                                         |    -    |
|         `backoff_config`          |  map   | Configures how to retry requests to Loki when a request fails. Default backoff schedule: 0.5s, 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s(4.267m). For a total time of 511.5s(8.5m) before logs are lost                                                                        |    -    |
|    `backoff_config.min_period`    | string | Initial backoff time between retries.                                                                                                                                                                                                                                          |  500ms  |
|    `backoff_config.max_period`    | string | Maximum backoff time between retries.                                                                                                                                                                                                                                          |   5m    |
|   `backoff_config.max_retries`    |  int   | Maximum number of retries to do.                                                                                                                                                                                                                                               |   10    |
|    `drop_rate_limited_batches`    |  bool  | Disable retries of batches that Loki responds to with a 429 status code (TooManyRequests). This reduces impacts on batches from other tenants, which could end up being delayed or dropped due to exponential backoff.                                                         |  false  |
|             `timeout`             | string | Maximum time to wait for a server to respond to a request                                                                                                                                                                                                                      |   10s   |



### example
A simple example:
```caddy
http://localhost:8080 {
	file_server browse
	log caddy {
		file_server browse
		format json
		output loki {
		    url http://example.com:3100/loki/api/v1/push
            tenant_id 1
            basic_auth{
                username admin
                password admin
            }
            labels {
                hostname {system.hostname}
                job web
            }
		}
	}
}
```
A full but invalid(full so there are filed conflicts) example:
```caddy
http://localhost:8080 {
	file_server browse
	log caddy {
		file_server browse
		format json
		output loki {
		    url http://example.com:3100/loki/api/v1/push
	        headers {
		        key1 value1
                key2 value2
	        }

	        tenant_id 1
	        batchwait 1s
	        batchsize 1048576

	        basic_auth {
		        username joshua
		        username_file /data/username
		        password replaceme
		        password_file /data/password
	        }

	        oauth2 {
		        client_id caddy-logger-loki
		        client_secret 114514
		        scopes profile
		        token_url https://sso.example.com
		        endpoint_params {
			        key1 value1
			        key2 value2
		        }
	        }

	        bearer_token 114514
	        bearer_token_file /data/token
	        proxy_url
	        tls_config {
		        ca_file /data/ca
		        cert_file /data/cert
		        key_file /data/key
		        server_name example.com
		        insecure_skip_verify false
	        }
	        backoff_config {
		        min_period 500ms
		        max_period 5m
		        max_retries 10
	        }
	        drop_rate_limited_batches false
	        labels {
		        key1 value1
		        key2 value2 
	        }
	        timeout 10s
	        max_streams 100
	        max_line_size 1024
	        max_line_size_truncate 1024
		}
	}
}
```