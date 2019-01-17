package backend

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/vault/logical"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest"
	"strconv"
	"testing"
)

func testGetBackend(t *testing.T) logical.Backend {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to initialize backend factory. %s", err)
	}

	return b
}

func buildConnURL(m map[string]interface{}) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		m["username"],
		m["password"],
		m["host"],
		m["port"],
		m["database"])
}

func prepareTestContainer(t *testing.T) (cleanup func(), attr map[string]interface{}) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Failed to connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "9.6.11", []string{"POSTGRES_PASSWORD=secret", "POSTGRES_DB=postgres"})
	if err != nil {
		t.Fatalf("Could not start local PostgreSQL docker container: %s", err)
	}

	cleanup = func() {
		err := pool.Purge(resource)
		if err != nil {
			t.Fatalf("Failed to cleanup local container: %s", err)
		}
	}

	port, err := strconv.Atoi(resource.GetPort("5432/tcp"))
	if err != nil {
		t.Fatalf("failed to parse the container port as integer. err: %s", err)
	}

	attr = map[string]interface{}{
		"username": "postgres",
		"password": "secret",
		"host":     "127.0.0.1",
		"port":     port,
		"database": "postgres",
		"ssl_mode": "disable",
	}

	if err = pool.Retry(func() error {
		var err error
		var db *sql.DB
		db, err = sql.Open("postgres", buildConnURL(attr))
		if err != nil {
			return err
		}
		defer db.Close()
		return db.Ping()
	}); err != nil {
		cleanup()
		t.Fatalf("Could not connect to PostgreSQL docker container: %s", err)
	}

	return
}
