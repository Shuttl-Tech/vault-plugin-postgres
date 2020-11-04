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

func TestBackend_cluster_basic(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attr := prepareTestContainer(t)
	defer cleanup()

	expectAttr := make(map[string]interface{})
	for k, v := range attr {
		expectAttr[k] = v
	}
	delete(expectAttr, "password")

	expectKeys := []string{
		"port", "max_open_connections", "max_idle_connections",
		"max_connection_lifetime", "database", "management_role",
		"host", "username", "password", "disabled", "ssl_mode",
	}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, "cluster/test-acc-cluster", attr, false),
			testAccReadClusterConfig(t, "cluster/test-acc-cluster", expectAttr, expectKeys, false),
			testAccDeleteClusterConfig(t, "cluster/test-acc-cluster", false),

			// Operating on a deleted cluster is an error
			testAccReadClusterConfig(t, "cluster/test-acc-cluster", nil, nil, true),
			testAccWriteClusterConfig(t, "cluster/test-acc-cluster", nil, true),
			testAccDeleteClusterConfig(t, "cluster/test-acc-cluster", true),

			// Can't access a cluster that is not registered
			testAccReadClusterConfig(t, "cluster/invalid-name", nil, nil, true),
		},
	})
}

func TestBackend_cluster_list(t *testing.T) {
	backend := testGetBackend(t)
	cleanup1, attr1 := prepareTestContainer(t)
	defer cleanup1()

	cleanup2, attr2 := prepareTestContainer(t)
	defer cleanup2()

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, "cluster/test-acc-cluster-one", attr1, false),
			testAccWriteClusterConfig(t, "cluster/test-acc-cluster-two", attr2, false),

			// Create databases in cluster
			testAccWriteDbConfig(t, "cluster/test-acc-cluster-one/test-db"),
			testAccWriteDbConfig(t, "cluster/test-acc-cluster-two/test-db"),

			testAccListClusters(t, "cluster/", "test-acc-cluster-one", "test-acc-cluster-two"),
		},
	})
}

func TestBackend_cluster_init(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attr := prepareTestContainer(t)
	defer cleanup()

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, "cluster/test-acc-init", attr, false),
			testAccValidateClusterInit(t, "cluster/test-acc-init"),
		},
	})
}

func TestBackend_DeleteCluster_Should_DeleteDatabase(t *testing.T) {
	backend := testGetBackend(t)
	cleanup1, attr1 := prepareTestContainer(t)
	defer cleanup1()

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, "cluster/test-acc-cluster-one", attr1, false),

			// Create databases in cluster
			testAccWriteDbConfig(t, "cluster/test-acc-cluster-one/test-db-one"),
			testAccWriteDbConfig(t, "cluster/test-acc-cluster-one/test-db-two"),
			testAccListDatabases(t, "cluster/test-acc-cluster-one", "test-db-one", "test-db-two"),

			testAccDeleteClusterConfig(t, "cluster/test-acc-cluster-one", false),
			testAccListDatabasesWithoutKeys(t, "cluster/test-acc-cluster-one"),
			testAccListDatabases(t, "gc/cluster/test-acc-cluster-one", "test-db-one", "test-db-two"),
		},
	})
}

func testAccValidateClusterInit(t *testing.T, target string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      target,
		ErrorOk:   true,
		Check:     testAccCheckValidClusterInit,
	}
}

func testAccCheckValidClusterInit(resp *logical.Response) error {
	if resp.IsError() {
		return fmt.Errorf("expected a valid response, received error: %s", resp.Error())
	}

	c := &ClusterConfig{}
	if err := mapstructure.Decode(resp.Data, c); err != nil {
		return err
	}

	e := func(n, e string) error {
		return fmt.Errorf("%s: %s", n, e)
	}

	// Check the credentials for root user
	rootDsn := c.dsn(connTypeRoot)
	dbRoot, err := sql.Open("postgres", rootDsn)
	if err != nil {
		return e("root credentials (open "+rootDsn+")", err.Error())
	}

	if err = dbRoot.Ping(); err != nil {
		return e("root credentials (ping "+rootDsn+")", err.Error())
	}

	if err = dbRoot.Close(); err != nil {
		return e("root credentials (close "+rootDsn+")", err.Error())
	}

	// Check the credentials for management user
	mgmtDsn := c.dsn(connTypeMgmt)
	dbMgmt, err := sql.Open("postgres", mgmtDsn)
	if err != nil {
		return e("management credentials (open "+mgmtDsn+")", err.Error())
	}

	if err = dbMgmt.Ping(); err != nil {
		return e("management credentials (ping "+mgmtDsn+")", err.Error())
	}

	if err = dbMgmt.Close(); err != nil {
		return e("management credentials (close "+mgmtDsn+")", err.Error())
	}

	return nil
}

func testAccReadClusterConfig(t *testing.T, target string, expect map[string]interface{}, expectKeys []string, expectError bool) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      target,
		ErrorOk:   true,
		Check: func(resp *logical.Response) error {
			if expectError {
				return checkErrResponse(resp)
			}

			if expectKeys != nil {
				for _, k := range expectKeys {
					if _, ok := resp.Data[k]; !ok {
						return fmt.Errorf("expected key %q to be present in response data, key not found", k)
					}
				}
			}

			if expect != nil {
				for k, ev := range expect {
					pv, ok := resp.Data[k]
					if !ok {
						return fmt.Errorf("expected key %q to be present in response data, key not found", k)
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

func testAccWriteClusterConfig(t *testing.T, target string, d map[string]interface{}, expectError bool) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      target,
		Data:      d,
		ErrorOk:   expectError,
		Check: func(resp *logical.Response) error {
			if expectError {
				return checkErrResponse(resp)
			}

			if resp == nil {
				return fmt.Errorf("expected a non-nil response")
			}

			if resp.IsError() {
				return fmt.Errorf("got an error response: %v", resp.Error())
			}

			return nil
		},
	}
}

func testAccDeleteClusterConfig(t *testing.T, target string, expectError bool) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.DeleteOperation,
		Path:      target,
		ErrorOk:   true,
		Check: func(resp *logical.Response) error {
			if expectError {
				return checkErrResponse(resp)
			} else if resp != nil && resp.IsError() {
				return fmt.Errorf("got an error response: %v", resp.Error())
			}

			return nil
		},
	}
}

func testAccListClusters(t *testing.T, target string, clusters ...string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ListOperation,
		Path:      target,
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			keys, ok := resp.Data["keys"]
			if !ok {
				return fmt.Errorf("expected keys attribute to exist in response")
			}

			if !reflect.DeepEqual(clusters, keys) {
				return fmt.Errorf("expected keys %+v, got %+v", clusters, keys)
			}

			return nil
		},
	}
}

func testAccListDatabasesWithoutKeys(t *testing.T, target string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ListOperation,
		Path:      target,
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			_, ok := resp.Data["keys"]
			if ok {
				return fmt.Errorf("not expected keys attributes to exist in response")
			}

			return nil
		},
	}
}

func checkErrResponse(resp *logical.Response) error {
	if resp.Data == nil {
		return fmt.Errorf("data is nil")
	}

	var e struct {
		Error string `mapstructure:"error"`
	}

	if err := mapstructure.Decode(resp.Data, &e); err != nil {
		return err
	}

	if len(e.Error) == 0 {
		return fmt.Errorf("expected error, but write succeeded")
	}

	return nil
}
