package backend

import (
	"database/sql"
	"fmt"
	logicaltest "github.com/hashicorp/vault/helper/testhelpers/logical"
	"github.com/hashicorp/vault/sdk/logical"
	_ "github.com/lib/pq"
	"github.com/mitchellh/mapstructure"
	"reflect"
	"testing"
)

func TestAccDatabaseCreate_basic(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attr := prepareTestContainer(t)
	defer cleanup()

	cluster := &ClusterConfig{}
	expectAttr := map[string]interface{}{
		"cluster":  "test-acc-db",
		"database": "test-db",
		"disabled": false,
	}

	expectKeys := []string{"objects_owner"}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, "cluster/test-acc-db", attr, false),
			testAccWriteDbConfig(t, "cluster/test-acc-db/test-db"),
			testAccReadDbConfig(t, "cluster/test-acc-db/test-db", expectAttr, expectKeys, false),
			testAccReadClusterConfigVar(t, "cluster/test-acc-db", cluster),
			testAccValidateDbInit(t, "cluster/test-acc-db/test-db", cluster),
			testAccDeleteDbConfig(t, "cluster/test-acc-db/test-db"),
			testAccReadDbConfig(t, "cluster/test-acc-db/test-db", nil, nil, true),
		},
	})
}

func TestAccDatabasesList(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attr := prepareTestContainer(t)
	defer cleanup()

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, "cluster/test-acc-db", attr, false),
			testAccWriteDbConfig(t, "cluster/test-acc-db/test-db-one"),
			testAccWriteDbConfig(t, "cluster/test-acc-db/test-db-two"),
			testAccListDatabases(t, "cluster/test-acc-db", "test-db-one", "test-db-two"),
		},
	})
}

func testAccListDatabases(t *testing.T, target string, dbs ...string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ListOperation,
		Path:      target,
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			keys, ok := resp.Data["keys"]
			if !ok {
				return fmt.Errorf("expected keys attributes to exist in response")
			}

			if !reflect.DeepEqual(keys, dbs) {
				return fmt.Errorf("expected keys %+v, got %+v", dbs, keys)
			}

			return nil
		},
	}
}

func testAccDeleteDbConfig(t *testing.T, name string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.DeleteOperation,
		Path:      name,
		ErrorOk:   false,
	}
}

func testAccReadDbConfig(t *testing.T, target string, expect map[string]interface{}, expectKeys []string, expectErr bool) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      target,
		ErrorOk:   true,
		Check: func(resp *logical.Response) error {
			if expectErr {
				return checkErrResponse(resp)
			}

			if expectKeys != nil {
				for _, k := range expectKeys {
					if _, ok := resp.Data[k]; !ok {
						return fmt.Errorf("expected key %q to be present in response %+v", k, resp.Data)
					}
				}
			}

			if expect != nil {
				for k, ev := range expect {
					pv, ok := resp.Data[k]
					if !ok {
						return fmt.Errorf("expected mapping %q to be present in response %+v", k, resp.Data)
					}

					if !reflect.DeepEqual(ev, pv) {
						return fmt.Errorf("value on response attribute %q does not match. expected %#v, found %#v", k, ev, pv)
					}
				}
			}

			return nil
		},
	}
}

func testAccReadClusterConfigVar(t *testing.T, name string, target *ClusterConfig) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      name,
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			if resp.IsError() {
				return resp.Error()
			}

			return mapstructure.Decode(resp.Data, target)
		},
	}
}

func testAccReadClusterConfigCallback(t *testing.T, name string, cb func(*ClusterConfig) error) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      name,
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			if resp.IsError() {
				return resp.Error()
			}

			ce := &ClusterConfig{}
			err := mapstructure.Decode(resp.Data, ce)
			if err != nil {
				return err
			}

			return cb(ce)
		},
	}
}

func testAccWriteDbConfig(t *testing.T, target string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.CreateOperation,
		Path:      target,
		ErrorOk:   false,
	}
}

func testAccValidateDbInit(t *testing.T, target string, cluster *ClusterConfig) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      target,
		ErrorOk:   false,
		Check:     testAccCheckValidDbInit(cluster),
	}
}

func testAccCheckValidDbInit(cluster *ClusterConfig) logicaltest.TestCheckFunc {
	return func(resp *logical.Response) error {
		if resp.IsError() {
			return resp.Error()
		}

		if resp.Data == nil {
			return fmt.Errorf("expected non-empty response, got nil")
		}

		db := &DbConfig{}
		err := mapstructure.Decode(resp.Data, db)
		if err != nil {
			return fmt.Errorf("failed to decode database configuration. %s", err)
		}

		conn, err := sql.Open("postgres", cluster.dsn(connTypeRoot))
		if err != nil {
			return fmt.Errorf("failed to open database connection. %s", err)
		}

		if err = conn.Ping(); err != nil {
			return fmt.Errorf("failed to ping database. %s", err)
		}
		defer conn.Close()

		// Check the database exists in cluster
		var dbExists bool
		err = conn.QueryRow(`select exists (select datname from pg_database where datname = $1)`, db.Database).Scan(&dbExists)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if !dbExists {
			return fmt.Errorf("database %q does no exist in cluster", db.Database)
		}

		// Check the objects owner role exists in cluster
		var ownerExists bool
		err = conn.QueryRow(`select exists (select rolname from pg_roles where rolname=$1)`, db.ObjectsOwner).Scan(&ownerExists)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if !ownerExists {
			return fmt.Errorf("role %q does not exist in cluster", db.ObjectsOwner)
		}

		return nil
	}
}
