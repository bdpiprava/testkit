# Testkit - A Go Integration Testing Framework

The `testkit` library is a Go integration testing framework that provides utilities to set up and tear down around
following:

- PostgreSQL database
- Kafka broker
- Elasticsearch cluster
- Http server

## Prerequisites

- Go (version 1.16 or later)
- PostgreSQL

## Configuration

The project uses a configuration file `.testkit.config.yml` to set up the PostgreSQL connection and logging level.

### Example Configuration

```yaml
---
log_level: trace

# PostgreSQL connection configuration
postgres:
  host: localhost:5432
  user: testkit
  password: badger
  database: testkit_db
  query_params:
    sslmode: disable

# Go migration configuration
go-migrate:
  database_name: template1
  migration_path: path/to/migrations
  fresh: true
  is_template: false
```

### Configuration Fields

| Field      | Description                                           |
|------------|-------------------------------------------------------|
| log_level  | Log level for the testkit library. Default is `info`. |
| postgres   | PostgreSQL connection configuration.                  |
| go-migrate | Go migration configuration.                           |

#### PostgreSQL Configuration Fields

This is the configuration for the PostgreSQL connection. Ideally connection details should be provided in the
configuration file. The `query_params` field is optional and can be used to provide additional query parameters for the
PostgreSQL connection.

| Field        | Description                                               |
|--------------|-----------------------------------------------------------|
| host         | PostgreSQL host and port.                                 |
| user         | PostgreSQL user.                                          |
| password     | PostgreSQL password.                                      |
| database     | PostgreSQL database name.                                 |
| query_params | Additional query parameters for the PostgreSQL connection |

#### Go Migration Configuration Fields

This is the configuration for the Go migration tool. If configured, the migration tool will run the migrations based on
configuration.

| Field          | Description                                               |
|----------------|-----------------------------------------------------------|
| database_name  | PostgreSQL database name.                                 |
| migration_path | Path to the directory containing migration files.         |
| fresh          | Recreate the database if exist before running migrations. |
| is_template    | Create the database as a template database.               |
