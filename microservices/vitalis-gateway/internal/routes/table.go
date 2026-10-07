package routes

import "strings"

type Access int

const (
	Public        Access = iota // não exige JWT
	Authenticated               // exige access token válido
)

// Route liga um prefixo de URL a um serviço interno.
type Route struct {
	Prefix   string
	Upstream string // chave em config.Upstreams
	Access   Access
	Roles    []string // vazio = qualquer usuário autenticado; senão exige ao menos uma
}

// Table espelha o x-routing-table do OpenAPI do Gateway.
// A ordem importa: prefixos mais específicos antes dos genéricos.
var Table = []Route{
	// Identity
	{Prefix: "/v1/auth", Upstream: "identity", Access: Public},
	{Prefix: "/v1/usuarios", Upstream: "identity", Access: Authenticated},
	{Prefix: "/v1/admin/usuarios", Upstream: "identity", Access: Authenticated, Roles: []string{"admin"}},

	// Webhook do provedor de pagamento: sem JWT; o Billing valida a assinatura HMAC.
	{Prefix: "/v1/webhooks/pagamentos", Upstream: "billing", Access: Public},

	// Billing
	{Prefix: "/v1/payment-intents", Upstream: "billing", Access: Authenticated},
	{Prefix: "/v1/subscriptions", Upstream: "billing", Access: Authenticated},
	{Prefix: "/v1/wallets", Upstream: "billing", Access: Authenticated},
	{Prefix: "/v1/invoices", Upstream: "billing", Access: Authenticated},
	{Prefix: "/v1/settlements", Upstream: "billing", Access: Authenticated},
	{Prefix: "/v1/commissions", Upstream: "billing", Access: Authenticated},

	// Clinical
	{Prefix: "/v1/especialidades", Upstream: "clinical", Access: Authenticated},
	{Prefix: "/v1/medicos", Upstream: "clinical", Access: Authenticated},
	{Prefix: "/v1/disponibilidade", Upstream: "clinical", Access: Authenticated},
	{Prefix: "/v1/consultas", Upstream: "clinical", Access: Authenticated},
	{Prefix: "/v1/triagens", Upstream: "clinical", Access: Authenticated},
	{Prefix: "/v1/videochamadas", Upstream: "clinical", Access: Authenticated},
	{Prefix: "/v1/prescricoes", Upstream: "clinical", Access: Authenticated},

	// Commerce
	{Prefix: "/v1/farmacias", Upstream: "commerce", Access: Authenticated},
	{Prefix: "/v1/categorias", Upstream: "commerce", Access: Authenticated},
	{Prefix: "/v1/produtos", Upstream: "commerce", Access: Authenticated},
	{Prefix: "/v1/estoque", Upstream: "commerce", Access: Authenticated},

	// Orders / Delivery
	{Prefix: "/v1/pedidos", Upstream: "orders", Access: Authenticated},
	{Prefix: "/v1/entregas", Upstream: "delivery", Access: Authenticated},
	{Prefix: "/v1/motoboys", Upstream: "delivery", Access: Authenticated},

	// Comms (inclui o WebSocket /v1/realtime)
	{Prefix: "/v1/notificacoes", Upstream: "comms", Access: Authenticated},
	{Prefix: "/v1/chat", Upstream: "comms", Access: Authenticated},
	{Prefix: "/v1/realtime", Upstream: "comms", Access: Authenticated},

	// Support / Files / Search / Audit
	{Prefix: "/v1/tickets", Upstream: "support", Access: Authenticated},
	{Prefix: "/v1/artigos-ajuda", Upstream: "support", Access: Authenticated},
	{Prefix: "/v1/objetos", Upstream: "files", Access: Authenticated},
	{Prefix: "/v1/pdf", Upstream: "files", Access: Authenticated},
	{Prefix: "/v1/search", Upstream: "search", Access: Authenticated},
	{Prefix: "/v1/audit", Upstream: "audit", Access: Authenticated, Roles: []string{"admin"}},
}

// Match devolve a primeira rota cujo prefixo casa com o path.
// "/v1/medicos" casa com "/v1/medicos" e "/v1/medicos/123", mas não com "/v1/medicosX".
func Match(path string) (Route, bool) {
	for _, rt := range Table {
		if path == rt.Prefix || strings.HasPrefix(path, rt.Prefix+"/") {
			return rt, true
		}
	}
	return Route{}, false
}
