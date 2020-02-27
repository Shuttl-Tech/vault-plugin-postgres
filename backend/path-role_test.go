package backend

import (
	"fmt"
	logicaltest "github.com/hashicorp/vault/helper/testhelpers/logical"
	"github.com/hashicorp/vault/sdk/logical"
	"reflect"
	"testing"
)

func TestAccRole_basic(t *testing.T) {
	backend := testGetBackend(t)
	roleAttr := map[string]interface{}{
		"default_ttl":          300,
		"max_ttl":              600,
		"creation_statement":   []string{"create role \"test-acc\" with login password \"secret\""},
		"revocation_statement": []string{"drop role \"test-acc\""},
	}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteRoleConfig(t, "roles/test-acc", roleAttr, false),
			testAccWriteRoleConfig(t, "roles/test-acc-one", roleAttr, false),
			testAccWriteRoleConfig(t, "roles/test-acc-two", roleAttr, false),
			testAccReadRoleConfig(t, "roles/test-acc", roleAttr, nil, false),
			testAccDeleteRoleConfig(t, "roles/test-acc", false),
			testAccListRolesConfig(t, "roles", []string{"test-acc-one", "test-acc-two"}),
		},
	})
}

func testAccListRolesConfig(t *testing.T, target string, expect []string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ListOperation,
		Path:      target,
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			keys, ok := resp.Data["keys"]
			if !ok {
				return fmt.Errorf("expected keys attribute to exist in response")
			}

			if !reflect.DeepEqual(expect, keys) {
				return fmt.Errorf("expected keys %v, got %v", expect, keys)
			}

			return nil
		},
	}
}

func testAccWriteRoleConfig(t *testing.T, target string, d map[string]interface{}, expectError bool) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      target,
		Data:      d,
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

func testAccReadRoleConfig(t *testing.T, target string, expect map[string]interface{}, expectKeys []string, expectError bool) logicaltest.TestStep {
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

func testAccDeleteRoleConfig(t *testing.T, target string, expectError bool) logicaltest.TestStep {
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
