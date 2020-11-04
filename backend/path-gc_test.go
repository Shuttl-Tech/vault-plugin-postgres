package backend

import (
	logicaltest "github.com/hashicorp/vault/helper/testhelpers/logical"
	_ "github.com/lib/pq"
	"testing"
)

func TestAccGCListClusters(t *testing.T) {
	backend := testGetBackend(t)

	cleanup1, attr1 := prepareTestContainer(t)
	cleanup2, attr2 := prepareTestContainer(t)

	defer cleanup1()
	defer cleanup2()

	expectedGc := []string{
		"test-cluster-one",
	}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			// Create two clusters
			testAccWriteClusterConfig(t, "cluster/test-cluster-one", attr1, false),
			testAccWriteClusterConfig(t, "cluster/test-cluster-two", attr2, false),

			// Create database in cluster
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db"),

			// Delete one cluster
			testAccDeleteClusterConfig(t, "cluster/test-cluster-one", false),

			// Assert that only the deleted cluster is marked for GC
			testAccListClusters(t, "gc/clusters", expectedGc...),
		},
	})
}

func TestAccGCGetCluster(t *testing.T) {
	backend := testGetBackend(t)

	cleanup1, attr1 := prepareTestContainer(t)
	cleanup2, attr2 := prepareTestContainer(t)

	defer cleanup1()
	defer cleanup2()

	expectAttr := make(map[string]interface{})
	for k, v := range attr1 {
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
			// Create two clusters
			testAccWriteClusterConfig(t, "cluster/test-cluster-one", attr1, false),
			testAccWriteClusterConfig(t, "cluster/test-cluster-two", attr2, false),

			// Delete one cluster
			testAccDeleteClusterConfig(t, "cluster/test-cluster-one", false),

			// Assert that only the deleted cluster is available on gc endpoint
			testAccReadClusterConfig(t, "gc/cluster/test-cluster-one", expectAttr, expectKeys, false),

			// Assert that the active cluster is not available on gc endpoint
			testAccReadClusterConfig(t, "gc/cluster/test-cluster-two", nil, nil, true),
		},
	})
}

func TestAccGcListDatabases(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attr := prepareTestContainer(t)
	defer cleanup()

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			// Create a cluster
			testAccWriteClusterConfig(t, "cluster/test-cluster-one", attr, false),

			// Create databases in the cluster
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-one"),
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-two"),
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-three"),

			// Delete databases
			testAccDeleteDbConfig(t, "cluster/test-cluster-one/test-db-one"),
			testAccDeleteDbConfig(t, "cluster/test-cluster-one/test-db-three"),

			// Assert that only deleted databases are marked for GC
			testAccListDatabases(t, "gc/cluster/test-cluster-one", "test-db-one", "test-db-three"),
		},
	})
}

func TestAccGcGetDatabase(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attr := prepareTestContainer(t)
	defer cleanup()

	expect := map[string]interface{}{
		"cluster":  "test-cluster-one",
		"database": "test-db-one",
		"disabled": true,
	}

	expectKeysGc := []string{
		"cluster", "database", "disabled", "objects_owner",
	}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			// Create a cluster
			testAccWriteClusterConfig(t, "cluster/test-cluster-one", attr, false),

			// Create databases in the cluster
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-one"),
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-two"),

			// Delete databases
			testAccDeleteDbConfig(t, "cluster/test-cluster-one/test-db-one"),

			// Validate that the deleted db is available for GC
			testAccReadDbConfig(t, "gc/cluster/test-cluster-one/test-db-one", expect, expectKeysGc, false),

			// Validate that the active db is not available for GC
			testAccReadDbConfig(t, "gc/cluster/test-cluster-one/test-db-two", nil, nil, true),
		},
	})
}

func TestGcPurgeDatabase(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attr := prepareTestContainer(t)
	defer cleanup()

	expectDbTwo := map[string]interface{}{
		"cluster":  "test-cluster-one",
		"database": "test-db-two",
		"disabled": false,
	}

	expectDbThree := map[string]interface{}{
		"cluster":  "test-cluster-one",
		"database": "test-db-three",
		"disabled": false,
	}

	expectKeys := []string{
		"cluster", "database", "disabled", "objects_owner",
	}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			// Create a cluster
			testAccWriteClusterConfig(t, "cluster/test-cluster-one", attr, false),

			// Create databases in the cluster
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-one"),
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-two"),
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-three"),

			// Delete databases
			testAccDeleteDbConfig(t, "cluster/test-cluster-one/test-db-one"),

			// Purge database configuration
			testAccDeleteDbConfig(t, "gc/cluster/test-cluster-one/test-db-one"),

			// Validate that the database has been purged
			testAccReadDbConfig(t, "cluster/test-cluster-one/test-db-one", nil, nil, true),
			testAccReadDbConfig(t, "gc/cluster/test-cluster-one/test-db-one", nil, nil, true),

			// Validate that the other databases are still available
			testAccReadDbConfig(t, "cluster/test-cluster-one/test-db-two", expectDbTwo, expectKeys, false),
			testAccReadDbConfig(t, "cluster/test-cluster-one/test-db-three", expectDbThree, expectKeys, false),
		},
	})
}

func TestGcPurgeCluster(t *testing.T) {
	backend := testGetBackend(t)

	cleanup1, attr1 := prepareTestContainer(t)
	cleanup2, attr2 := prepareTestContainer(t)

	defer cleanup1()
	defer cleanup2()

	expectAttr := make(map[string]interface{})
	for k, v := range attr2 {
		expectAttr[k] = v
	}
	delete(expectAttr, "password")

	dbKeysMap := func(c, d string, e bool) map[string]interface{} {
		return map[string]interface{}{
			"cluster":  "test-cluster-" + c,
			"database": "test-db-" + d,
			"disabled": e,
		}
	}

	expectDbKeys := []string{
		"cluster", "database", "disabled", "objects_owner",
	}

	expectClusterKeys := []string{
		"port", "max_open_connections", "max_idle_connections",
		"max_connection_lifetime", "database", "management_role",
		"host", "username", "password", "disabled", "ssl_mode",
	}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			// Create two clusters
			testAccWriteClusterConfig(t, "cluster/test-cluster-one", attr1, false),
			testAccWriteClusterConfig(t, "cluster/test-cluster-two", attr2, false),

			// Create databases in both clusters
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-one"),
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-two"),
			testAccWriteDbConfig(t, "cluster/test-cluster-one/test-db-three"),
			testAccWriteDbConfig(t, "cluster/test-cluster-two/test-db-one"),
			testAccWriteDbConfig(t, "cluster/test-cluster-two/test-db-two"),
			testAccWriteDbConfig(t, "cluster/test-cluster-two/test-db-three"),

			// Delete cluster one
			testAccDeleteClusterConfig(t, "cluster/test-cluster-one", false),

			// Assert that databases are marked for GC after delete cluster
			testAccListDatabases(t, "gc/cluster/test-cluster-one", "test-db-one", "test-db-three", "test-db-two"),

			// Purge deleted cluster
			testAccDeleteClusterConfig(t, "gc/cluster/test-cluster-one", false),

			// Assert that databases are purged after purge cluster
			testAccListDatabasesWithoutKeys(t, "gc/cluster/test-cluster-one"),

			// Assert that the purged cluster is no longer available
			testAccReadClusterConfig(t, "gc/cluster/test-cluster-one", nil, nil, true),

			// Assert that the database of the purged cluster are also purged
			testAccReadDbConfig(t, "cluster/test-cluster-one/test-db-one", nil, nil, true),
			testAccReadDbConfig(t, "gc/cluster/test-cluster-one/test-db-one", nil, nil, true),
			testAccReadDbConfig(t, "cluster/test-cluster-one/test-db-two", nil, nil, true),
			testAccReadDbConfig(t, "gc/cluster/test-cluster-one/test-db-two", nil, nil, true),
			testAccReadDbConfig(t, "cluster/test-cluster-one/test-db-three", nil, nil, true),
			testAccReadDbConfig(t, "gc/cluster/test-cluster-one/test-db-three", nil, nil, true),

			// Assert that the active cluster is still available
			testAccReadClusterConfig(t, "cluster/test-cluster-two", expectAttr, expectClusterKeys, false),

			// Assert that the databases in active clusters are also available
			testAccReadDbConfig(t, "cluster/test-cluster-two/test-db-one", dbKeysMap("two", "one", false), expectDbKeys, false),
			testAccReadDbConfig(t, "cluster/test-cluster-two/test-db-two", dbKeysMap("two", "two", false), expectDbKeys, false),
			testAccReadDbConfig(t, "cluster/test-cluster-two/test-db-three", dbKeysMap("two", "three", false), expectDbKeys, false),
		},
	})
}
