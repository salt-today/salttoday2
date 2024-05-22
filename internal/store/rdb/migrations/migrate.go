package migrations

import (
	"database/sql"
	"embed"
	"io/fs"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
)

//go:embed *.sql
var sources embed.FS

func MigrateDb(db *sql.DB) error {
	entry := logrus.WithField(`component`, `sql-storage-migration`)
	files, err := getAllFilenames(&sources)
	if err != nil {
		panic(err)
	}
	entry.WithField(`files`, files).Info(`migration files found`)
	migrations := migrate.EmbedFileSystemMigrationSource{
		FileSystem: sources,
		Root:       ".",
	}
	n, err := migrate.Exec(db, `mysql`, migrations, migrate.Up)
	if err != nil {
		entry.WithError(err).Fatal(`unable to migrate DB`)
		return err
	}
	entry.WithField(`migrations`, n).Info(`migrations applied successfully`)

	return nil
}

func getAllFilenames(efs *embed.FS) (files []string, err error) {
	if err := fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}
