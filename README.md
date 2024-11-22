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

## Installation

```shell
go get github.com/bdpiprava/testkit@latest
```

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

# Elasticsearch connection configuration
elasticsearch:
  addresses: http://localhost:9200 # comma separated list of addresses
  username: testkit
  password: badger

# APIMock configuration
api-mock:
  address: http://localhost:8080
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

#### Elasticsearch Configuration Fields

This is the configuration for the Elasticsearch connection.

| Field     | Description                                      |
|-----------|--------------------------------------------------|
| addresses | Comma separated list of Elasticsearch addresses. |
| username  | Elasticsearch username.                          |
| password  | Elasticsearch password.                          |

#### APIMock Configuration Fields

This is the configuration for the APIMock server. It uses wiremock to mock the API responses.

| Field   | Description                     |
|---------|---------------------------------|
| address | Address of the wiremock server. |

## Usage

The `testkit` library provides a `Suite` struct that can be embedded in the test suite struct. The `Suite` struct
provides
the following methods to set up and tear down the resources:

### PostgreSQL Helper Methods

- **RequiresPostgresDatabase** - Sets up a PostgreSQL database and returns a `*sqlx.DB` connection.

### Kafka Helper Methods

- **RequiresKafka** - Sets up a Kafka cluster and returns the server address.
- **Produce** - Produces a message to the Kafka topic.
- **Consume** - Consumes a message from the Kafka topic on message read callback function is called. Return `true` from
  callback function to stop consuming messages.

### Elasticsearch Helper Methods

- **CreateIndex** - Creates an Elasticsearch index with the given name and other params.
- **DeleteIndex** - Deletes the Elasticsearch index.
- **IndexExists** - Checks if the Elasticsearch index exists.
- **CloseIndices** - Closes the Elasticsearch indices.
- **FindIndices** - Finds the Elasticsearch indices.
- **GetIndexSettings** - Gets the settings of the Elasticsearch index.
- **EventuallyBlockStatus** - Checks the index block for the given index.

### APIMock Helper Methods

- **SetupAPIMocksFromFile** - Sets up the services mock from a file and returns the URLs. It takes the file path and a
  map of dynamic parameters as parameters to replace the template values in the file.

### Example

```go
package example_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bdpiprava/testkit"
)

type ExampleTestSuite struct {
	testkit.Suite
}

func TestDatabaseIntegrationTestSuite(t *testing.T) {
	testkit.Run(t, new(ExampleTestSuite))
}

func (s *ExampleTestSuite) TestSuite_ExampleTest() {
	db := s.RequiresPostgresDatabase("test")

	var version string
	err := db.Get(&version, "SELECT VERSION()")
	s.Require().NoError(err)

	s.Require().NotEmpty(version)
	s.Contains(version, "PostgreSQL")
}
```