![Provision](https://raw.githubusercontent.com/txn2/provision/master/mast.jpg)
[![Provision Release](https://img.shields.io/github/release/txn2/provision.svg)](https://github.com/txn2/provision/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/txn2/provision)](https://goreportcard.com/report/github.com/txn2/provision)
[![GoDoc](https://godoc.org/github.com/txn2/query?status.svg)](https://godoc.org/github.com/txn2/provision)
[![Docker Container Image Size](https://shields.beevelop.com/docker/image/image-size/txn2/provision/latest.svg)](https://hub.docker.com/r/txn2/provision/)
[![Docker Container Layers](https://shields.beevelop.com/docker/image/layers/txn2/provision/latest.svg)](https://hub.docker.com/r/txn2/provision/)

**Provision** is a user and account micro-platform, a highly opinionated building block for TXN2 components. **Provision** defines basic object models that represent the foundation for an account and user. **Provision** is intended as a fundamental dependency of current and future TXN2 platform services.

## Configuration

Configuration is inherited from [txn2/micro](https://github.com/txn2/micro#configuration). The
following configuration is specific to **provision**:

| Flag          | Environment Variable | Description                                                |
|:--------------|:---------------------|:-----------------------------------------------------------|
| -esServer     | ELASTIC_SERVER       | Elasticsearch Server (default "http://elasticsearch:9200") |
| -systemPrefix | SYSTEM_PREFIX        | Prefix for system indices. (default "system_")             |

## Development

Testing using Elasticsearch and Kibana in docker compose:
```bash
docker-compose up
```

Run for source:
```bash
go run ./cmd/provisison.go --esServer="http://localhost:9200"
```

## Examples

### Util

Get prefix:
```bash
curl http://localhost:8080/prefix
```

### Account

Upsert account:
```bash
curl -X POST \
  http://localhost:8080/account \
  -d '{
	"id": "xorg",
	"description": "Organization X is an IOT data collection agency.",
	"display_name": "Organization X",
	"active": true,
    "modules": [
        "telematics",
        "wx",
        "data_science",
        "gpu"
    ]
}'
```

Get account:
```bash
curl http://localhost:8080/account/xorg
```

Search accounts:
```bash
curl -X POST \
  http://localhost:8080/searchAccounts \
  -d '{
  "query": {
    "match_all": {}
  }
}'
```

### User

Upsert user:
```bash
curl -X POST \
  http://localhost:8080/user \
  -d '{
	"id": "sysop",
	"description": "Global system operator",
	"display_name": "System Operator",
	"active": true,
	"sysop": true,
	"password": "examplepassword",
	"sections_all": false,
	"sections": [],
	"accounts": [],
	"admin_accounts": []
}'
```

Get user:
```bash
curl http://localhost:8080/user/sysop
```

Search users:
```bash
curl -X POST \
  http://localhost:8080/searchUsers \
  -d '{
  "query": {
    "match_all": {}
  }
}'
```

Authenticate user:
```bash
curl -X POST \
  http://localhost:8080/authUser \
  -d '{
	"id": "sysop",
	"password": "examplepassword"
}'
```

Access check:
```bash
# first get a token
TOKEN=$(curl -s -X POST \
          http://localhost:8080/authUser?raw=true \
          -d '{
        	"id": "sysop",
        	"password": "examplepassword"
        }') && echo $TOKEN
        
# check for basic access
curl -X POST \
  http://localhost:8080/userHasAccess \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
	"sections": ["a","b"],
	"accounts": ["example","example2"]
}'

# check for admin access
curl -X POST \
  http://localhost:8080/userHasAdminAccess \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
	"sections": ["a","b"],
	"accounts": ["example","example2"]
}'
```

## Release Packaging

Build test release:
```bash
goreleaser --skip-publish --rm-dist --skip-validate
```

Build and release:
```bash
GITHUB_TOKEN=$GITHUB_TOKEN goreleaser --rm-dist
```
