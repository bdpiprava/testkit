package testkit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bdpiprava/testkit"
	"github.com/bdpiprava/testkit/internal"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

const (
	configWithoutGoMigrate = `---
log_level: error

# PostgreSQL connection configuration
postgres:
  host: localhost:5544
  user: testkit
  password: badger
  database: testkit_db
  query_params:
    sslmode: disable
`
)

func Test_InitialiseDatabase_MissingConfigFile(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")
	_ = os.Setenv(internal.EnvConfigLocation, "non-existing-config.yml")

	db, err := testkit.InitialiseDatabase()

	assert.Nil(t, db)
	assert.EqualError(t, err, "open non-existing-config.yml: no such file or directory")
}

func Test_InitialiseDatabase_MissingGoMigrateConfigInFile(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")

	file := createFile(t, "config.yaml", configWithoutGoMigrate)
	_ = os.Setenv(internal.EnvConfigLocation, file)

	db, err := testkit.InitialiseDatabase()

	assert.Nil(t, db)
	assert.EqualError(t, err, testkit.ErrMissingGoMigrateConfig.Error())
}

func Test_InitialiseDatabase_WithTemplateTrue(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")

	migrationDir := createTestMigration(t)
	file := createGoMigrateConfig(t, migrationDir, "template_001", true, true)
	_ = os.Setenv(internal.EnvConfigLocation, file)

	// When
	db, err := testkit.InitialiseDatabase()

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assertDatabaseCreated(t, db, "template_001", true)
	defer closeSilently(db)
}

func Test_InitialiseDatabase_WithTemplateFalse(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")

	migrationDir := createTestMigration(t)
	file := createGoMigrateConfig(t, migrationDir, "template_002", false, true)
	_ = os.Setenv(internal.EnvConfigLocation, file)

	// When
	db, err := testkit.InitialiseDatabase()

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assertDatabaseCreated(t, db, "template_002", false)
	defer closeSilently(db)
}

func closeSilently(db *sqlx.DB) {
	_ = db.Close()
}

func createFileInDir(name, content string) error {
	return os.WriteFile(name, []byte(content), 0644)
}

func createFile(t *testing.T, name, content string) string {
	file := filepath.Join(t.TempDir(), name)
	assert.NoError(t, os.WriteFile(file, []byte(content), 0644))
	return file
}

func createGoMigrateConfig(t *testing.T, migrationPath, database string, template, fresh bool) string {
	content := fmt.Sprintf(`---
log_level: error

# PostgreSQL connection configuration
postgres:
  host: localhost:5544
  user: testkit
  password: badger
  database: testkit_db
  query_params:
    sslmode: disable
go-migrate:
  database_name: %s
  migration_path: %s
  fresh: %t
  is_template: %t
`, database, migrationPath, fresh, template)

	return createFile(t, "config.yaml", content)
}

func createTestMigration(t *testing.T) string {
	rootDir := filepath.Join(t.TempDir(), "migrations")
	assert.NoError(t, os.MkdirAll(rootDir, 0755))

	assert.NoError(t, createFileInDir(filepath.Join(rootDir, "001_test_migration.up.sql"), "CREATE TABLE test_table (id serial PRIMARY KEY);"))
	assert.NoError(t, createFileInDir(filepath.Join(rootDir, "001_test_migration.down.sql"), "DROP TABLE test_table;"))

	return rootDir
}

func unsetEnvVar() {
	_ = os.Unsetenv(internal.EnvConfigLocation)
	_ = os.Unsetenv(internal.EnvDisableConfigCache)
}

func assertDatabaseCreated(t *testing.T, db *sqlx.DB, database string, isTemplate bool) {
	var fromDB bool
	assert.NoError(t, db.Get(&fromDB, fmt.Sprintf(`SELECT datistemplate FROM pg_database WHERE datname='%s'`, database)))
	assert.Equal(t, fromDB, isTemplate)
}
