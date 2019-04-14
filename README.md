![Provision](https://raw.githubusercontent.com/txn2/provision/master/mast.jpg)

**Provision** is a user and account micro-platform, a highly opinionated building block for TXN2 components. **Provision** defines basic object models that represent the foundation for an account and user. **Provision** is intended as a fundamental dependency of current and future TXN2 platform services.

## Configuration

Configuration is inherited from [txn2/ack](https://github.com/txn2/ack#configuration). The
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

### Account

#### Upsert Account

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

#### Get Account

```bash
curl http://localhost:8080/account/xorg
```

### User

#### Upsert User

```bash
curl -X POST \
  http://localhost:8080/user \
  -d '{
	"id": "sysop",
	"description": "Global system operator",
	"display_name": "System Operator",
	"active": true,
	"sysop": true,
	"password": "JntGihgkGRvFKmb74XgUwn7bcYpb3bPtWE8gEVTpvChEqpsFwq",
	"sections_all": false,
	"sections": [],
	"accounts": [],
	"admin_accounts": []
}'
```

#### Get User

```bash
curl http://localhost:8080/user/sysop
```
