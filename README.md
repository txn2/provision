![Provision](https://raw.githubusercontent.com/txn2/provision/master/mast.jpg)
[![Provision Release](https://img.shields.io/github/release/txn2/provision.svg)](https://github.com/txn2/provision/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/txn2/provision)](https://goreportcard.com/report/github.com/txn2/provision)
[![GoDoc](https://godoc.org/github.com/txn2/provision?status.svg)](https://godoc.org/github.com/txn2/provision)
[![Docker Container Image Size](https://shields.beevelop.com/docker/image/image-size/txn2/provision/latest.svg)](https://hub.docker.com/r/txn2/provision/)
[![Docker Container Layers](https://shields.beevelop.com/docker/image/layers/txn2/provision/latest.svg)](https://hub.docker.com/r/txn2/provision/)

**Provision** is a user and account micro-platform, a highly opinionated building block for TXN2 components. **Provision** defines basic object models that represent the foundation for an account, user and asset. **Provision** is intended as a fundamental dependency of current and future TXN2 platform services.

- Elasticsearch is used as a database for **[Account]**, **[User]** and **[Asset]** objects.
- Intended for basic storage, retrieval and searching.


**Provision** is intended as in internal service to be accessed by other services. Use a secure
reverse proxy for direct access by system operators.

## Configuration

Configuration is inherited from [txn2/micro](https://github.com/txn2/micro#configuration). The
following configuration is specific to **provision**:

| Flag          | Environment Variable | Description                                                |
|:--------------|:---------------------|:-----------------------------------------------------------|
| -esServer     | ELASTIC_SERVER       | Elasticsearch Server (default "http://elasticsearch:9200") |
| -systemPrefix | SYSTEM_PREFIX        | Prefix for system indices. (default "system_")             |

## Routes

| Method | Route Pattern                                             | Description                                                               |
|:-------|:----------------------------------------------------------|:--------------------------------------------------------------------------|
| GET    | [/prefix](#get-prefix)                                    | Get the prefix used for Elasticsearch indexes.                            |
| POST   | [/account](#upsert-account)                               | Upsert an Account object.                                                 |
| GET    | [/account/:id](#get-account)                              | Get an Account ojbect by id.                                              |
| POST   | [/keyCheck/:id](#check-key)                               | Check if an AccessKey is associated with an account.                      |
| POST   | [/searchAccounts](#search-accounts)                       | Search for Accounts with a Lucene query.                                  |
| POST   | [/user](#upsert-user)                                     | Upsert a User object.                                                     |
| GET    | [/user/:id](#get-user)                                    | Get a User object by id.                                                  |
| POST   | [/searchUsers](#search-users)                             | Search for Users with a Lucene query.                                     |
| POST   | [/userHasAccess](#access-check)                           | Post an AccessCheck object with Token to determine basic access.          |
| POST   | [/userHasAdminAccess](#access-check)                      | Post an AccessCheck object with Token to determine admin access.          |
| POST   | [/authUser](#authenticate-user)                           | Post Credentials and if valid receive a Token.                            |
| POST   | [/asset](#upsert-asset)                                   | Upsert an Asset.                                                          |
| GET    | [/asset/:id](#get-asset)                                  | Get an asset by id.                                                       |
| POST   | [/searchAssets](#search-assets)                           | Search for Assets with a Lucene query.                                    |
| GET    | /adm/:parentAccount/account/:account                      | Get a child account.                                                      |
| POST   | /adm/:parentAccount/account                               | Upsert a child account.                                                   |
| GET    | /adm/:parentAccount/children                              | Get children of parent account.                                           |
| GET    | /adm/:parentAccount/assets/:account                       | Get assets with associations to account.                                  |
| GET    | /adm/:parrentId/assetAssoc/:asset/:accountFrom/:accountTo | Re-associate any routes from specified account to another (child or self) |


## Development

Testing using Elasticsearch and Kibana in docker compose:
```bash
docker-compose up
```

Run for source:
```bash
go run ./cmd/provision.go --esServer="http://localhost:9200"
```

## Examples

### Util

#### Get Prefix
```bash
curl http://localhost:8080/prefix
```

### Account

#### Upsert Account
```bash
curl -X POST \
  http://localhost:8080/account \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test_account",
    "description": "This is a test account",
    "display_name": "Test Organization",
    "active": true,
    "access_keys": [
        {
            "name": "test-data",
            "key": "sRqhFPdudA9s8qtVqgixHXyU8ubbYhrCBttC8amLdMwkxeZHskseNXyCRe4eXRxP",
            "description": "Generic access key",
            "active": true
        },
        {
            "name": "test",
            "key": "PDWgYr3bQGNoLptBRDkLTGQcRmCMqLGRFpXoXJ8xMPsMLMg3LHvWpJgDu2v3LYBA",
            "description": "Generic access key 2",
            "active": true
        }
    ],
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
curl http://localhost:8080/account/test_account
```

#### Search Accounts
```bash
curl -X POST \
  http://localhost:8080/searchAccounts \
  -d '{
  "query": {
    "match_all": {}
  }
}'
```

#### Check Key
```bash
curl -X POST \
  http://localhost:8080/keyCheck/test_account \
  -H 'Content-Type: application/json' \
  -d '{ 
	"name": "test_data", 
	"key": "sRqhFPdudA9s8qtVqgixHXyU8ubbYhrCBttC8amLdMwkxeZHskseNXyCRe4eXRxP"
}'
```

### User

#### Upsert User
```bash
curl -X POST \
  http://localhost:8080/user \
  -H 'Content-Type: application/json' \
  -d '{
	"id": "test_user",
	"description": "Test User non-admin",
	"display_name": "Test User",
	"active": true,
	"sysop": false,
	"password": "eWidL7UtiWJABHgn8WAv8MWbqNKjHUqhNC7ZaWotEFKYNrLvzAwwCXC9eskPFJoY",
	"sections_all": false,
	"sections": ["api", "config", "data"],
	"accounts": ["test"],
	"admin_accounts": []
}'
```

#### Get User
```bash
curl -X GET http://localhost:8080/user/test_user
```

#### Search Users
```bash
curl -X POST \
  http://localhost:8080/searchUsers \
  -d '{
  "query": {
    "match_all": {}
  }
}'
```

#### Authenticate User
```bash
curl -X POST \
  http://localhost:8080/authUser \
  -H 'Content-Type: application/json' \
  -d '{
	"id": "test_user",
	"password": "eWidL7UtiWJABHgn8WAv8MWbqNKjHUqhNC7ZaWotEFKYNrLvzAwwCXC9eskPFJoY"
}'
```

#### Access Check
```bash
# first get a token
TOKEN=$(curl -s -X POST \
          http://localhost:8080/authUser?raw=true \
          -d '{
        	"id": "test_user",
        	"password": "eWidL7UtiWJABHgn8WAv8MWbqNKjHUqhNC7ZaWotEFKYNrLvzAwwCXC9eskPFJoY"
        }') && echo $TOKEN
        
# check for basic access
curl -X POST \
  http://localhost:8080/userHasAccess \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
	"sections": ["api"],
	"accounts": ["test"]
}'

# check for admin access
curl -X POST \
  http://localhost:8080/userHasAdminAccess \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
	"sections": ["api"],
	"accounts": ["test"]
}'
```

### Asset

#### Upsert Asset
```bash
curl -X POST \
  http://localhost:8080/asset \
  -H 'Content-Type: application/json' \
  -d '{
	"id": "test-unique-asset-id-12345",
	"description": "A unique asset in the system.",
	"display_name": "Asset 12345",
	"active": true,
	"asset_class": "iot_device",
	"routes": [
		{ "account_id": "test", "model_id": "device_details", type: "system" },
		{ "account_id": "test", "model_id": "device_location", type: "account" }
	]
}'
```

#### Get Asset
```bash
curl -X GET http://localhost:8080/asset/test-unique-asset-id-12345
```

#### Search Assets
```bash
curl -X POST \
  http://localhost:8080/searchAssets \
  -H 'Content-Type: application/json' \
  -d '{
  "query": {
    "match_all": {}
  }
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

[Account]: https://godoc.org/github.com/txn2/provision#Account
[User]: https://godoc.org/github.com/txn2/provision#User
[Asset]: https://godoc.org/github.com/txn2/provision#Asset