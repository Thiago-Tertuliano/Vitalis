from pathlib import Path

root = Path(r"D:\Carreira\Thiago Matos Tertuliano - Carreira\Projetos\Vitalis\docs\contracts\openapi")
root.mkdir(parents=True, exist_ok=True)

COMMON = """
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
  parameters:
    IdempotencyKey:
      name: Idempotency-Key
      in: header
      required: false
      schema: { type: string }
      description: Obrigatorio em POST de cobranca/assinatura (Billing).
    RequestId:
      name: X-Request-Id
      in: header
      required: false
      schema: { type: string }
  schemas:
    Error:
      type: object
      required: [erro, codigo]
      properties:
        erro: { type: string }
        codigo: { type: string }
        request_id: { type: string }
        detalhes:
          type: object
          additionalProperties: true
    Health:
      type: object
      properties:
        status: { type: string, enum: [ok, degraded] }
        service: { type: string }
"""

HEALTH = """
  /health:
    get:
      tags: [health]
      summary: Liveness
      security: []
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Health'
  /ready:
    get:
      tags: [health]
      summary: Readiness
      security: []
      responses:
        '200': { description: Ready }
        '503': { description: Not ready }
"""


def header(title: str, port: int, desc: str) -> str:
    return f"""openapi: 3.0.3
info:
  title: {title}
  version: 0.1.0
  description: |
    {desc}

    Contrato canonico Vitalis-contracts. Clientes publicos via Gateway :8080.
servers:
  - url: http://localhost:{port}
    description: Servico direto (rede interna)
  - url: http://localhost:8080
    description: Via Gateway
tags:
  - name: health
  - name: domain
security:
  - bearerAuth: []
"""


def write(name: str, body: str) -> None:
    (root / name).write_text(body, encoding="utf-8")
    print("wrote", name)


write(
    "gateway.yaml",
    header("Vitalis Gateway", 8080, "Ponto unico de entrada. Roteia /v1/* e WS; sem dominio de negocio.")
    + "paths:"
    + HEALTH
    + """
  /v1/realtime:
    get:
      tags: [domain]
      summary: WebSocket proxy -> Comms
      responses:
        '101': { description: Switching Protocols }
  /v1/webhooks/pagamentos:
    post:
      tags: [domain]
      summary: Webhook * API Pagamentos -> Billing
      security: []
      responses:
        '200': { description: Encaminhado }
x-routing-table:
  /v1/auth/*: identity:8081
  /v1/usuarios/*: identity:8081
  /v1/payment-intents/*: billing:8086
  /v1/subscriptions/*: billing:8086
  /v1/wallets/*: billing:8086
  /v1/invoices/*: billing:8086
  /v1/settlements/*: billing:8086
  /v1/commissions/*: billing:8086
  /v1/especialidades: clinical:8082
  /v1/medicos: clinical:8082
  /v1/disponibilidade: clinical:8082
  /v1/consultas/*: clinical:8082
  /v1/triagens/*: clinical:8082
  /v1/videochamadas/*: clinical:8082
  /v1/prescricoes/*: clinical:8082
  /v1/farmacias/*: commerce:8083
  /v1/categorias: commerce:8083
  /v1/produtos/*: commerce:8083
  /v1/estoque/*: commerce:8083
  /v1/pedidos/*: orders:8084
  /v1/entregas/*: delivery:8085
  /v1/motoboys/*: delivery:8085
  /v1/notificacoes/*: comms:8087
  /v1/chat/*: comms:8087
  /v1/tickets/*: support:8088
  /v1/artigos-ajuda/*: support:8088
  /v1/objetos/*: files:8089
  /v1/pdf/*: files:8089
  /v1/search: search:8090
  /v1/audit/*: audit:8091
"""
    + COMMON,
)

write(
    "identity.yaml",
    header("Vitalis Identity API", 8081, "Autenticacao, usuarios, papeis e enderecos.")
    + "paths:"
    + HEALTH
    + """
  /v1/auth/login:
    post:
      tags: [domain]
      summary: Login
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email, senha]
              properties:
                email: { type: string, format: email }
                senha: { type: string, format: password }
      responses:
        '200':
          description: Tokens emitidos
          content:
            application/json:
              schema:
                type: object
                properties:
                  access_token: { type: string }
                  token_type: { type: string, example: Bearer }
                  expires_in: { type: integer }
                  user:
                    type: object
                    properties:
                      id: { type: string }
                      nome: { type: string }
                      email: { type: string }
                      roles:
                        type: array
                        items: { type: string }
        '401':
          description: Credenciais invalidas
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
  /v1/auth/refresh:
    post:
      tags: [domain]
      summary: Renovar access token (refresh rotacionado)
      security: []
      responses:
        '200': { description: Novo access token }
        '401': { description: Refresh invalido/revogado }
  /v1/auth/logout:
    post:
      tags: [domain]
      summary: Revoga refresh token
      responses:
        '204': { description: Logout OK }
  /v1/auth/registro:
    post:
      tags: [domain]
      summary: Cadastro de usuario
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [nome, email, senha, tipo_usuario]
              properties:
                nome: { type: string }
                email: { type: string, format: email }
                senha: { type: string }
                telefone: { type: string }
                tipo_usuario:
                  type: string
                  enum: [paciente, medico, farmacia, motoboy, admin]
                cpf: { type: string }
                data_nascimento: { type: string, format: date }
      responses:
        '201': { description: Criado }
        '409': { description: Email ja existe }
  /v1/auth/esqueci-senha:
    post:
      tags: [domain]
      summary: Solicitar reset de senha
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email]
              properties:
                email: { type: string, format: email }
      responses:
        '202': { description: Aceito }
  /v1/auth/redefinir-senha:
    post:
      tags: [domain]
      summary: Redefinir senha com token
      security: []
      responses:
        '204': { description: Senha alterada }
  /v1/auth/jwks.json:
    get:
      tags: [domain]
      summary: JWKS publico RS256
      security: []
      responses:
        '200': { description: JWK Set }
  /v1/usuarios/me:
    get:
      tags: [domain]
      summary: Perfil do usuario autenticado
      responses:
        '200': { description: Perfil }
    put:
      tags: [domain]
      summary: Atualizar perfil
      responses:
        '200': { description: Atualizado }
  /v1/usuarios/me/enderecos:
    get:
      tags: [domain]
      summary: Listar enderecos
      responses:
        '200': { description: Lista }
    post:
      tags: [domain]
      summary: Criar endereco
      responses:
        '201': { description: Criado }
  /v1/usuarios/me/enderecos/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema: { type: string }
    put:
      tags: [domain]
      summary: Atualizar endereco
      responses:
        '200': { description: OK }
    delete:
      tags: [domain]
      summary: Remover endereco
      responses:
        '204': { description: Removido }
  /v1/admin/usuarios/{id}/roles:
    parameters:
      - name: id
        in: path
        required: true
        schema: { type: string }
    put:
      tags: [domain]
      summary: Alterar papeis (admin)
      responses:
        '200': { description: Roles atualizadas }
        '403': { description: Sem permissao }
"""
    + COMMON,
)

write(
    "billing.yaml",
    header("Vitalis Billing API", 8086, "PIX, cartao, carteira, assinaturas, ledger e split.")
    + "paths:"
    + HEALTH
    + """
  /v1/payment-intents:
    post:
      tags: [domain]
      summary: Criar PaymentIntent
      parameters:
        - $ref: '#/components/parameters/IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [valor_centavos, meio, reference_type, reference_id]
              properties:
                valor_centavos: { type: integer, minimum: 1 }
                moeda: { type: string, default: BRL }
                meio: { type: string, enum: [pix, cartao, carteira] }
                reference_type: { type: string, enum: [subscription, order, wallet_topup] }
                reference_id: { type: string }
      responses:
        '201': { description: Intent criado }
  /v1/payment-intents/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema: { type: string }
    get:
      tags: [domain]
      summary: Obter PaymentIntent
      responses:
        '200': { description: Intent }
  /v1/payment-intents/{id}/pix:
    parameters:
      - name: id
        in: path
        required: true
        schema: { type: string }
      - $ref: '#/components/parameters/IdempotencyKey'
    post:
      tags: [domain]
      summary: Gerar cobranca PIX
      responses:
        '200': { description: QR / EMV / expires_at }
  /v1/payment-intents/{id}/cancel:
    parameters:
      - name: id
        in: path
        required: true
        schema: { type: string }
    post:
      tags: [domain]
      summary: Cancelar intent
      responses:
        '200': { description: Cancelado }
  /v1/payment-intents/{id}/refund:
    parameters:
      - name: id
        in: path
        required: true
        schema: { type: string }
      - $ref: '#/components/parameters/IdempotencyKey'
    post:
      tags: [domain]
      summary: Estornar parcial/total
      responses:
        '200': { description: Estorno iniciado }
  /v1/subscriptions:
    post:
      tags: [domain]
      summary: Assinar ou trocar plano
      parameters:
        - $ref: '#/components/parameters/IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [plano_id]
              properties:
                plano_id: { type: string }
                tipo_troca: { type: string, enum: [nova, upgrade, downgrade, troca] }
                meio: { type: string, enum: [pix, cartao, carteira] }
      responses:
        '201': { description: Assinatura processada }
  /v1/subscriptions/me:
    get:
      tags: [domain]
      summary: Minha assinatura ativa
      responses:
        '200': { description: Assinatura }
        '404': { description: Sem assinatura }
  /v1/subscriptions/{id}/cancel:
    parameters:
      - name: id
        in: path
        required: true
        schema: { type: string }
    post:
      tags: [domain]
      summary: Cancelar assinatura
      responses:
        '200': { description: Cancelada }
  /v1/invoices:
    get:
      tags: [domain]
      summary: Listar faturas
      responses:
        '200': { description: Lista }
  /v1/wallets/me:
    get:
      tags: [domain]
      summary: Saldo da carteira
      responses:
        '200': { description: Wallet }
  /v1/wallets/topup:
    post:
      tags: [domain]
      summary: Creditar carteira
      parameters:
        - $ref: '#/components/parameters/IdempotencyKey'
      responses:
        '201': { description: Top-up criado }
  /v1/settlements:
    get:
      tags: [domain]
      summary: Liquidacoes parceiro/admin
      responses:
        '200': { description: Lista }
  /v1/commissions:
    get:
      tags: [domain]
      summary: Comissoes do profissional
      responses:
        '200': { description: Lista }
  /v1/internal/subscriptions/active:
    get:
      tags: [domain]
      summary: Gate interno — assinatura ativa (Clinical)
      parameters:
        - name: user_id
          in: query
          required: true
          schema: { type: string }
      responses:
        '200': { description: Status assinatura }
  /v1/webhooks/pagamentos:
    post:
      tags: [domain]
      summary: Webhook * API Pagamentos
      security: []
      responses:
        '200': { description: Aceito }
        '401': { description: Assinatura invalida }
"""
    + COMMON.replace(
        "required: false\n      schema: { type: string }\n      description: Obrigatorio em POST de cobranca/assinatura (Billing).",
        "required: true\n      schema: { type: string }\n      description: Obrigatorio em POST de cobranca/assinatura.",
    ),
)

SERVICES = {
    "clinical.yaml": (
        8082,
        "Vitalis Clinical API",
        "Agenda, consultas, triagem, videochamada e prescricoes.",
        """
  /v1/especialidades:
    get:
      tags: [domain]
      summary: Listar especialidades
      security: []
      responses: { '200': { description: Lista } }
  /v1/medicos:
    get:
      tags: [domain]
      summary: Buscar medicos
      responses: { '200': { description: Lista } }
  /v1/disponibilidade:
    get:
      tags: [domain]
      summary: Slots disponiveis
      responses: { '200': { description: Slots } }
  /v1/consultas:
    get:
      tags: [domain]
      summary: Listar consultas do usuario
      responses: { '200': { description: Lista } }
    post:
      tags: [domain]
      summary: Agendar consulta (exige plano ativo)
      responses:
        '201': { description: Agendada }
        '402': { description: Plano inativo }
  /v1/consultas/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Detalhe da consulta
      responses: { '200': { description: Consulta }, '403': { description: Sem acesso } }
    patch:
      tags: [domain]
      summary: Atualizar status
      responses: { '200': { description: OK } }
  /v1/triagens:
    post:
      tags: [domain]
      summary: Enviar triagem
      responses: { '201': { description: Criada } }
  /v1/triagens/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Resultado triagem
      responses: { '200': { description: Triagem } }
  /v1/videochamadas/sala:
    post:
      tags: [domain]
      summary: Criar/entrar sala de video
      responses: { '201': { description: Sessao } }
  /v1/prescricoes:
    get:
      tags: [domain]
      summary: Listar prescricoes
      responses: { '200': { description: Lista } }
    post:
      tags: [domain]
      summary: Emitir prescricao (medico)
      responses: { '201': { description: Emitida }, '403': { description: Sem role medico } }
  /v1/prescricoes/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Detalhe prescricao
      responses: { '200': { description: Prescricao } }
""",
    ),
    "commerce.yaml": (
        8083,
        "Vitalis Commerce API",
        "Farmacias, catalogo e estoque.",
        """
  /v1/farmacias:
    get:
      tags: [domain]
      summary: Listar farmacias
      responses: { '200': { description: Lista } }
    post:
      tags: [domain]
      summary: Cadastrar farmacia
      responses: { '201': { description: Criada } }
  /v1/farmacias/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Detalhe farmacia
      responses: { '200': { description: Farmacia } }
  /v1/categorias:
    get:
      tags: [domain]
      summary: Categorias
      responses: { '200': { description: Lista } }
  /v1/produtos:
    get:
      tags: [domain]
      summary: Catalogo
      responses: { '200': { description: Lista } }
    post:
      tags: [domain]
      summary: Criar produto (farmacia)
      responses: { '201': { description: Criado } }
  /v1/produtos/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Detalhe produto
      responses: { '200': { description: Produto } }
    put:
      tags: [domain]
      summary: Atualizar produto
      responses: { '200': { description: OK } }
  /v1/estoque/reservar:
    post:
      tags: [domain]
      summary: Reservar estoque (Orders)
      responses: { '200': { description: Reservado }, '409': { description: Sem estoque } }
  /v1/estoque/liberar:
    post:
      tags: [domain]
      summary: Liberar reserva
      responses: { '200': { description: Liberado } }
""",
    ),
    "orders.yaml": (
        8084,
        "Vitalis Orders API",
        "Pedidos de farmacia.",
        """
  /v1/pedidos:
    get:
      tags: [domain]
      summary: Listar pedidos
      responses: { '200': { description: Lista } }
    post:
      tags: [domain]
      summary: Criar pedido
      responses: { '201': { description: Criado } }
  /v1/pedidos/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Detalhe pedido
      responses: { '200': { description: Pedido }, '403': { description: Sem acesso } }
  /v1/pedidos/{id}/cancelar:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    post:
      tags: [domain]
      summary: Cancelar pedido
      responses: { '200': { description: Cancelado } }
  /v1/pedidos/{id}/status:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    patch:
      tags: [domain]
      summary: Atualizar status
      responses: { '200': { description: OK } }
""",
    ),
    "delivery.yaml": (
        8085,
        "Vitalis Delivery API",
        "Motoboy, alocacao e rastreamento.",
        """
  /v1/motoboys/me:
    get:
      tags: [domain]
      summary: Perfil do motoboy
      responses: { '200': { description: Courier } }
  /v1/motoboys/me/status:
    patch:
      tags: [domain]
      summary: online/offline/busy
      responses: { '200': { description: OK } }
  /v1/entregas:
    get:
      tags: [domain]
      summary: Listar entregas
      responses: { '200': { description: Lista } }
  /v1/entregas/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Detalhe entrega
      responses: { '200': { description: Entrega } }
  /v1/entregas/{id}/aceitar:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    post:
      tags: [domain]
      summary: Aceitar corrida
      responses: { '200': { description: Assigned } }
  /v1/entregas/{id}/status:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    patch:
      tags: [domain]
      summary: Atualizar status
      responses: { '200': { description: OK } }
  /v1/entregas/{id}/rastreamento:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Pontos de rastreio
      responses: { '200': { description: Tracking } }
    post:
      tags: [domain]
      summary: Enviar ponto GPS
      responses: { '201': { description: Ponto registrado } }
""",
    ),
    "comms.yaml": (
        8087,
        "Vitalis Comms API",
        "Notificacoes, chat e WebSocket.",
        """
  /v1/notificacoes:
    get:
      tags: [domain]
      summary: Listar notificacoes
      responses: { '200': { description: Lista } }
  /v1/notificacoes/{id}/lida:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    patch:
      tags: [domain]
      summary: Marcar como lida
      responses: { '200': { description: OK } }
  /v1/chat/salas:
    get:
      tags: [domain]
      summary: Listar salas
      responses: { '200': { description: Lista } }
    post:
      tags: [domain]
      summary: Criar sala
      responses: { '201': { description: Criada } }
  /v1/chat/salas/{id}/mensagens:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Historico
      responses: { '200': { description: Mensagens } }
    post:
      tags: [domain]
      summary: Enviar mensagem
      responses: { '201': { description: Enviada } }
  /v1/realtime:
    get:
      tags: [domain]
      summary: Endpoint WS
      responses: { '101': { description: Upgrade } }
""",
    ),
    "support.yaml": (
        8088,
        "Vitalis Support API",
        "Tickets e central de ajuda.",
        """
  /v1/tickets:
    get:
      tags: [domain]
      summary: Listar tickets
      responses: { '200': { description: Lista } }
    post:
      tags: [domain]
      summary: Abrir ticket
      responses: { '201': { description: Criado } }
  /v1/tickets/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Detalhe ticket
      responses: { '200': { description: Ticket }, '403': { description: Sem acesso } }
  /v1/tickets/{id}/comentarios:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    post:
      tags: [domain]
      summary: Comentar
      responses: { '201': { description: Comentario } }
  /v1/artigos-ajuda:
    get:
      tags: [domain]
      summary: Listar artigos
      security: []
      responses: { '200': { description: Lista } }
  /v1/artigos-ajuda/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Ler artigo
      security: []
      responses: { '200': { description: Artigo } }
""",
    ),
    "files.yaml": (
        8089,
        "Vitalis Files API",
        "Upload, URLs assinadas e PDF.",
        """
  /v1/objetos:
    post:
      tags: [domain]
      summary: Metadata + URL upload
      responses: { '201': { description: Upload URL } }
  /v1/objetos/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Metadata
      responses: { '200': { description: Objeto } }
  /v1/objetos/{id}/url-assinada:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    post:
      tags: [domain]
      summary: URL assinada download
      responses: { '200': { description: URL TTL curto } }
  /v1/objetos/{id}/confirmar:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    post:
      tags: [domain]
      summary: Confirmar upload
      responses: { '200': { description: Confirmado } }
  /v1/pdf/jobs:
    post:
      tags: [domain]
      summary: Enfileirar PDF
      responses: { '202': { description: Job criado } }
  /v1/pdf/jobs/{id}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string } }
    get:
      tags: [domain]
      summary: Status job PDF
      responses: { '200': { description: Job } }
""",
    ),
    "search.yaml": (
        8090,
        "Vitalis Search API",
        "Busca unificada (indice).",
        """
  /v1/search:
    get:
      tags: [domain]
      summary: Busca unificada
      parameters:
        - name: q
          in: query
          required: true
          schema: { type: string }
        - name: tipo
          in: query
          schema: { type: string, enum: [medico, produto, artigo, todos] }
      responses: { '200': { description: Resultados } }
""",
    ),
    "audit.yaml": (
        8091,
        "Vitalis Audit API",
        "Trilha LGPD append-only.",
        """
  /v1/audit/events:
    get:
      tags: [domain]
      summary: Consultar trilha (admin)
      parameters:
        - { name: actor_id, in: query, schema: { type: string } }
        - { name: resource_id, in: query, schema: { type: string } }
        - { name: from, in: query, schema: { type: string, format: date-time } }
        - { name: to, in: query, schema: { type: string, format: date-time } }
      responses: { '200': { description: Eventos }, '403': { description: Somente admin } }
  /v1/internal/audit/ingest:
    post:
      tags: [domain]
      summary: Ingestao interna
      responses: { '202': { description: Aceito } }
""",
    ),
}

for fname, (port, title, desc, paths) in SERVICES.items():
    write(fname, header(title, port, desc) + "paths:" + HEALTH + paths + COMMON)

print("total", len(list(root.glob('*.yaml'))))
