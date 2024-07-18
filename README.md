# Caddy Logger Loki Plugin

A plugin of caddy to push log directly to loki without promtail but compliable.

## Usage

### parameters:

to do

### example

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
		}
	}
}
```