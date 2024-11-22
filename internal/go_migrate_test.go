package internal_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/bdpiprava/testkit/internal"
)

func Test_InitialiseDatabase_WhenMissingGoMigrateConfig(t *testing.T) {
	db, err := internal.InitialiseDatabase(internal.SuiteConfig{}, logrus.NewEntry(logrus.New()))

	require.Nil(t, db)
	require.EqualError(t, err, internal.ErrMissingGoMigrateConfig.Error())
}

func Test_InitialiseDatabase_WithTemplateTrue(t *testing.T) {
	migrationDir := createTestMigration(t)

	// When
	db, err := internal.InitialiseDatabase(internal.SuiteConfig{
		GoMigrateConfig: &internal.GoMigrateConfig{
			DatabaseName:  "template_001",
			MigrationPath: migrationDir,
			IsTemplate:    true,
			Fresh:         true,
		},
		PostgresConfig: internal.PostgresConfig{
			Host:        "localhost:5544",
			User:        "testkit",
			Password:    "badger",
			Database:    "testkit_db",
			QueryParams: map[string]string{"sslmode": "disable"},
		},
	}, logrus.NewEntry(logrus.New()))

	// Then
	require.NoError(t, err)
	require.NotNil(t, db)
	assertDatabaseCreated(t, db, "template_001", true)
	defer closeSilently(db)
}

func Test_InitialiseDatabase_WithTemplateFalse(t *testing.T) {
	migrationDir := createTestMigration(t)

	// When
	db, err := internal.InitialiseDatabase(internal.SuiteConfig{
		GoMigrateConfig: &internal.GoMigrateConfig{
			DatabaseName:  "template_002",
			MigrationPath: migrationDir,
			IsTemplate:    false,
			Fresh:         true,
		},
		PostgresConfig: internal.PostgresConfig{
			Host:        "localhost:5544",
			User:        "testkit",
			Password:    "badger",
			Database:    "testkit_db",
			QueryParams: map[string]string{"sslmode": "disable"},
		},
	}, logrus.NewEntry(logrus.New()))

	// Then
	require.NoError(t, err)
	require.NotNil(t, db)
	assertDatabaseCreated(t, db, "template_002", false)
	defer closeSilently(db)
}

func closeSilently(db *sqlx.DB) {
	_ = db.Close()
}

func createFileInDir(name, content string) error {
	return os.WriteFile(name, []byte(content), 0600)
}

func createTestMigration(t *testing.T) string {
	rootDir := filepath.Join(t.TempDir(), "migrations")
	require.NoError(t, os.MkdirAll(rootDir, 0755))

	upContent := "CREATE TABLE test_table (id serial PRIMARY KEY);"
	downContent := "DROP TABLE test_table;"

	require.NoError(t, createFileInDir(filepath.Join(rootDir, "001_test_migration.up.sql"), upContent))
	require.NoError(t, createFileInDir(filepath.Join(rootDir, "001_test_migration.down.sql"), downContent))

	return rootDir
}

func assertDatabaseCreated(t *testing.T, db *sqlx.DB, database string, isTemplate bool) {
	var fromDB bool
	require.NoError(t, db.Get(&fromDB, fmt.Sprintf(`SELECT datistemplate FROM pg_database WHERE datname='%s'`, database)))
	require.Equal(t, fromDB, isTemplate)
}
