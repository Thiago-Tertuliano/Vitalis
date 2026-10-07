package routes

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		path     string
		ok       bool
		upstream string
	}{
		{"/v1/auth/login", true, "identity"},
		{"/v1/auth", true, "identity"},
		{"/v1/medicos/123/agenda", true, "clinical"},
		{"/v1/admin/usuarios/1/roles", true, "identity"},
		{"/v1/webhooks/pagamentos", true, "billing"},
		{"/v1/medicosX", false, ""},
		{"/v1/internal/users", false, ""},
		{"/v2/auth/login", false, ""},
		{"/", false, ""},
	}
	for _, tc := range cases {
		rt, ok := Match(tc.path)
		if ok != tc.ok || rt.Upstream != tc.upstream {
			t.Errorf("Match(%q) = (%q, %v), esperado (%q, %v)", tc.path, rt.Upstream, ok, tc.upstream, tc.ok)
		}
	}
}

func TestTable_SoAuthEWebhookSaoPublicos(t *testing.T) {
	publicas := map[string]bool{"/v1/auth": true, "/v1/webhooks/pagamentos": true}
	for _, rt := range Table {
		if rt.Access == Public && !publicas[rt.Prefix] {
			t.Errorf("rota %s está pública sem estar na lista permitida", rt.Prefix)
		}
	}
}
