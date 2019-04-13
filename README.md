![Provision](https://raw.githubusercontent.com/txn2/provision/master/mast.jpg)

**Provision** is a user and account micro-platform, a highly opinionated building block for TXN2 components. **Provision** defines basic object models that represent the foundation for an account and user. **Provision** is intended as a fundamental dependency of current and future TXN2 platform services.

## Development

Testing using Elasticsearch and Kibana in docker compose:
```bash
docker-compose up
```

Run for source:
```bash
go run ./cmd/provisison.go --esServer="http://localhost:9200"
```

View data in kibana:
- http://localhost:5601