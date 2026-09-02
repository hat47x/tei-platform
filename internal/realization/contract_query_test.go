package realization

import "testing"

func TestContractQueryTargetIdentityUsesDatabaseContractAndQuery(t *testing.T) {
	target := ContractQueryTarget{
		Database: "master_data",
		Contract: "master_data.customer",
		Query:    "find_by_id",
	}

	if got, want := target.Identity(), "master_data/master_data.customer/find_by_id"; got != want {
		t.Fatalf("Contract Query target identity=%q want=%q", got, want)
	}
}

func TestContractQueryTargetIdentityDoesNotIncludeApplicationOrProvider(t *testing.T) {
	first := ContractQueryTarget{
		Database: "master_data",
		Contract: "master_data.customer",
		Query:    "find_by_id",
	}
	second := ContractQueryTarget{
		Database: "master_data",
		Contract: "master_data.customer",
		Query:    "find_by_id",
	}

	if first.Identity() != second.Identity() {
		t.Fatalf("同じDatabase Contract Queryのtarget identityが実現側の文脈で変化しています: %q != %q", first.Identity(), second.Identity())
	}
}
