package repository

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestSupportMigrationLineEndingCompatibility(t *testing.T) {
	for _, name := range []string{"243_custom_support_tickets.sql", "244_custom_support_retention.sql"} {
		t.Run(name, func(t *testing.T) {
			content, err := migrations.FS.ReadFile(name)
			require.NoError(t, err)
			lf := strings.ReplaceAll(string(content), "\r\n", "\n")
			crlf := strings.ReplaceAll(lf, "\n", "\r\n")
			rule := migrationChecksumCompatibilityRules[name]
			require.Equal(t, rule.fileChecksum, migrationChecksum(lf))
			require.Contains(t, rule.acceptedDBChecksum, migrationChecksum(crlf))

			for _, tc := range []struct {
				name, filename, db, file string
				ok                       bool
			}{
				{"crlf_to_lf", name, migrationChecksum(crlf), lf, true},
				{"lf_to_crlf", name, migrationChecksum(lf), crlf, true},
				{"changed_sql", name, migrationChecksum(crlf), lf + "\nSELECT 1;", false},
				{"unknown_db", name, "unknown", lf, false},
				{"other_migration", "999_other.sql", migrationChecksum(crlf), lf, false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					db, mock, err := sqlmock.New()
					require.NoError(t, err)
					defer func() { _ = db.Close() }()
					prepareMigrationsBootstrapExpectations(mock)
					mock.ExpectQuery("SELECT checksum FROM schema_migrations WHERE filename = \\$1").
						WithArgs(tc.filename).
						WillReturnRows(sqlmock.NewRows([]string{"checksum"}).AddRow(tc.db))
					mock.ExpectExec("SELECT pg_advisory_unlock\\(\\$1\\)").
						WithArgs(migrationsAdvisoryLockID).
						WillReturnResult(sqlmock.NewResult(0, 1))
					// No SQL replay or checksum update is expected for an applied migration.
					err = applyMigrationsFS(context.Background(), db, fstest.MapFS{
						tc.filename: &fstest.MapFile{Data: []byte(tc.file)},
					})
					if tc.ok {
						require.NoError(t, err)
					} else {
						require.ErrorContains(t, err, "checksum mismatch")
					}
					require.NoError(t, mock.ExpectationsWereMet())
				})
			}
		})
	}
}
