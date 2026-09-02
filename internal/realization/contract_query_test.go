package realization

import "testing"

func TestContractQueryTargetIdentityUsesDatabaseContractAndQuery(t *testing.T) {
	target := ContractQueryTarget{
		Database: "master_data",
		Contract: "master_data.customer",
		Query:    "find_by_id",
	}

	if !target.Valid() {
		t.Fatalf("三要素が明示されたContract Query targetを不完全と判定しています: %#v", target)
	}
	if got, want := target.Identity(), "master_data/master_data.customer/find_by_id"; got != want {
		t.Fatalf("Contract Query target identity=%q want=%q", got, want)
	}
}

func TestContractQueryTargetRejectsIncompleteIdentity(t *testing.T) {
	tests := []struct {
		name   string
		target ContractQueryTarget
	}{
		{name: "database missing", target: ContractQueryTarget{Contract: "master_data.customer", Query: "find_by_id"}},
		{name: "contract missing", target: ContractQueryTarget{Database: "master_data", Query: "find_by_id"}},
		{name: "query missing", target: ContractQueryTarget{Database: "master_data", Contract: "master_data.customer"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.target.Valid() {
				t.Fatalf("不完全なContract Query targetを有効と判定しています: %#v", tt.target)
			}
		})
	}
}
