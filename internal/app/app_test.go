package app

import "testing"

func TestLegacyPACParams(t *testing.T) {
	uuid, typ := legacyPACParams(`/pacabc/YSozOntzOjE6ImgiO3M6ODoiYWJjZGVmMTIiO3M6MToidCI7czoyOiJzaSI7czoxOiJzIjtzOjM2OiIxMTExMTExMS0xMTExLTQxMTEtODExMS0xMTExMTExMTExMTEiO30=`)
	if uuid != "11111111-1111-4111-8111-111111111111" || typ != "si" {
		t.Fatalf("bad legacy params: %q %q", uuid, typ)
	}
}
