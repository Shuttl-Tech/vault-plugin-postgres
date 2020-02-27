package backend

import (
	"fmt"
	logicaltest "github.com/hashicorp/vault/helper/testhelpers/logical"
	"github.com/hashicorp/vault/sdk/logical"
	"reflect"
	"sort"
	"testing"
)

func TestMatchAttrs(t *testing.T) {
	sm := map[string]string{
		"key-one": "value-one",
	}

	tm := map[string]string{
		"key-one": "value-one",
		"key-two": "value-two",
	}

	if !matchAttrs(tm, sm) {
		t.Fatalf("Expected %+v to match %+v", sm, tm)
	}
}

func TestMetadata_basic(t *testing.T) {
	backend := testGetBackend(t)
	cleanup, attr := prepareTestContainer(t)
	defer cleanup()

	expectAttr := make(map[string]interface{})
	for k, v := range attr {
		expectAttr[k] = v
	}
	delete(expectAttr, "password")

	metadata := map[string]interface{}{
		"meta-one": "val-one",
		"meta-two": "val-two",
	}

	matcher := map[string]interface{}{
		"meta-one": "val-one",
	}

	expectListKeys := []string{
		"cluster/test-acc-cluster",
		"database/test-acc-cluster-test-db",
	}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, "cluster/test-acc-cluster", attr, false),
			testAccWriteDbConfig(t, "cluster/test-acc-cluster/test-db"),
			testAccWriteClusterMetadata(t, "test-acc-cluster", metadata),
			testAccWriteDbMetadata(t, "test-acc-cluster", "test-db", metadata),
			testAccFindClusterMatch(t, matcher, []string{"test-acc-cluster"}),
			testAccFindDbMatch(t, matcher, []string{"test-acc-cluster/test-db"}),
			testAccListMetadata(t, expectListKeys),
			testAccDeleteMeta(t, "cluster/test-acc-cluster"),
			testAccFindClusterMatch(t, matcher, nil),
			testAccDeleteMeta(t, "database/test-acc-cluster-test-db"),
			testAccFindDbMatch(t, matcher, nil),
		},
	})
}

func testAccWriteClusterMetadata(t *testing.T, cluster string, meta map[string]interface{}) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "metadata",
		ErrorOk:   false,
		Data: map[string]interface{}{
			"cluster": cluster,
			"data":    meta,
		},

		Check: func(resp *logical.Response) error {
			if _, ok := resp.Data["id"]; !ok {
				return fmt.Errorf("expected id attribute in response")
			}

			return nil
		},
	}
}

func testAccWriteDbMetadata(t *testing.T, cluster, db string, meta map[string]interface{}) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "metadata",
		ErrorOk:   false,
		Data: map[string]interface{}{
			"cluster":  cluster,
			"database": db,
			"data":     meta,
		},

		Check: func(resp *logical.Response) error {
			if _, ok := resp.Data["id"]; !ok {
				return fmt.Errorf("expected id attribute in response")
			}

			return nil
		},
	}
}

func testAccFindClusterMatch(t *testing.T, matcher map[string]interface{}, expect interface{}) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "metadata",
		ErrorOk:   false,
		Data: map[string]interface{}{
			"type": "cluster",
			"data": matcher,
		},

		Check: func(resp *logical.Response) error {
			if !reflect.DeepEqual(expect, resp.Data["keys"]) {
				return fmt.Errorf("expected %+v to match %+v", expect, resp.Data["keys"])
			}

			return nil
		},
	}
}

func testAccFindDbMatch(t *testing.T, matcher map[string]interface{}, expect interface{}) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "metadata",
		ErrorOk:   false,
		Data: map[string]interface{}{
			"type": "database",
			"data": matcher,
		},
		Check: func(resp *logical.Response) error {
			if !reflect.DeepEqual(expect, resp.Data["keys"]) {
				return fmt.Errorf("expected %+v to match %+v", expect, resp.Data["keys"])
			}

			return nil
		},
	}
}

func testAccListMetadata(t *testing.T, expect []string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ListOperation,
		Path:      "metadata",
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			keys := resp.Data["keys"].([]string)
			sort.Strings(expect)
			sort.Strings(keys)
			if !reflect.DeepEqual(expect, keys) {
				return fmt.Errorf("expected %+v to match %+v", expect, keys)
			}

			return nil
		},
	}
}

func testAccDeleteMeta(t *testing.T, id string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.DeleteOperation,
		Path:      "metadata/" + id,
		ErrorOk:   false,
	}
}
