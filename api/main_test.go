package api

import (
	"database/sql"
	"io/ioutil"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattes/migrate/source/file"
	log "github.com/sirupsen/logrus"

	"github.com/mattes/migrate"
	"github.com/mattes/migrate/database/postgres"
)

var testDB *sql.DB

func determineMigrationCount() int {
	files, err := ioutil.ReadDir("/migrations")
	if err != nil {
		log.Fatalf("missing test migrations files: %v", err)
	}

	migrationCount := 0
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			migrationCount++
		}
	}
	return migrationCount
}

// this function not only waits for the database to accept its incoming connection, but also performs any necessary migrations
func migrateDatabase(db *sql.DB, migrationCount int) {
	databaseIsNotMigrated := true
	for databaseIsNotMigrated {
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			log.Printf("waiting half a second for the test database")
			time.Sleep(500 * time.Millisecond)
		} else {
			m, err := migrate.NewWithDatabaseInstance("file:///migrations", "postgres", driver)
			if err != nil {
				log.Fatalf("error encountered setting up new test migration client: %v", err)
			}

			for i := 0; i < migrationCount; i++ {
				err = m.Steps(1)
				if err != nil {
					log.Printf("error encountered migrating test database: %v", err)
				}
			}
			databaseIsNotMigrated = false
		}
	}
}

func init() {
	// Connect to the database
	dbURL := os.Getenv("DAIRYCART_TEST_DB_URL")
	var err error
	testDB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("error encountered connecting to test database: %v", err)
	}
	migrateDatabase(testDB, determineMigrationCount())
}