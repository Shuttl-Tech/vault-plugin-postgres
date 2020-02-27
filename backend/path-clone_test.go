package backend

import (
	logicaltest "github.com/hashicorp/vault/helper/testhelpers/logical"
	"github.com/hashicorp/vault/sdk/logical"
	"testing"
)

func TestCloneUpdate(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attrs := prepareTestContainer(t)
	defer cleanup()

	var (
		testCloneSourceCluster = "test-clone-source"
		testCloneTargetCluster = "test-clone-target"
		testCloneDb            = "test-clone-db"
	)

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, "cluster/"+testCloneSourceCluster, attrs, false),
			testAccWriteDbConfig(t, "cluster/"+testCloneSourceCluster+"/"+testCloneDb),
			testAccCloneCluster(t, attrs, testCloneSourceCluster, testCloneTargetCluster),
		},
	})
}

func testAccCloneCluster(t *testing.T, attrs map[string]interface{}, sourceName, targetName string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "clone/" + sourceName,
		ErrorOk:   false,
		Data: map[string]interface{}{
			"target": targetName,
			"host":   attrs["host"].(string),
			"port":   attrs["port"].(int),
		},
		Check: func(resp *logical.Response) error {
			for _, w := range resp.Warnings {
				t.Logf("Received warning on clone: %s", w)
			}

			return nil
		},
	}
}
