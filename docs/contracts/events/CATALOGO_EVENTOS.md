# Catálogo de eventos — Vitalis (Async)

Envelope padrão (ADR-003):

```json
{
  "id": "evt_01J...",
  "type": "payment.succeeded",
  "occurred_at": "2026-09-12T12:00:00Z",
  "producer": "Vitalis-billing",
  "version": 1,
  "correlation_id": "req_...",
  "data": {}
}
```

Barramento: **Redis Streams**. Publicação via **Transactional Outbox**.

## Eventos

| type | Producer | Consumers | data (principais) |
|---|---|---|---|
| `user.registered` | identity | billing, clinical, audit, comms | `user_id`, `email`, `roles` |
| `user.updated` | identity | orders, delivery, comms | `user_id`, `changed_fields` |
| `user.role_changed` | identity | audit, comms | `user_id`, `roles` |
| `subscription.activated` | billing | clinical, comms | `user_id`, `subscription_id`, `plano_id` |
| `subscription.changed` | billing | clinical, comms | `user_id`, `plano_id`, `tipo_troca` |
| `subscription.cancelled` | billing | clinical, comms | `user_id`, `subscription_id` |
| `subscription.past_due` | billing | clinical, comms | `user_id`, `subscription_id` |
| `payment.intent.created` | billing | audit | `payment_intent_id`, `meio`, `valor_centavos` |
| `payment.succeeded` | billing | orders, delivery, comms, audit | `payment_intent_id`, `reference_type`, `reference_id` |
| `payment.failed` | billing | orders, comms | `payment_intent_id`, `motivo` |
| `payment.expired` | billing | orders, comms | `payment_intent_id` |
| `payment.refunded` | billing | orders, audit | `payment_intent_id`, `valor_centavos` |
| `split.allocated` | billing | audit | `reference_type`, `allocations[]` |
| `settlement.completed` | billing | comms, audit | `settlement_id`, `beneficiary` |
| `wallet.credited` | billing | comms | `user_id`, `valor_centavos` |
| `wallet.debited` | billing | comms | `user_id`, `valor_centavos` |
| `commission.calculated` | billing | comms | `medico_id`, `appointment_id`, `valor_centavos` |
| `appointment.created` | clinical | comms | `appointment_id`, `patient_id`, `doctor_id` |
| `appointment.confirmed` | clinical | comms | `appointment_id` |
| `appointment.cancelled` | clinical | comms | `appointment_id` |
| `appointment.completed` | clinical | billing, comms | `appointment_id`, `patient_id`, `doctor_id` |
| `triage.completed` | clinical | comms | `triage_id`, `patient_id` |
| `prescription.issued` | clinical | comms, files | `prescription_id`, `patient_id`, `doctor_id` |
| `video.session.started` | clinical | comms | `session_id`, `appointment_id` |
| `video.session.ended` | clinical | comms, audit | `session_id` |
| `product.updated` | commerce | search | `product_id` |
| `stock.changed` | commerce | orders | `product_id`, `pharmacy_id`, `qty` |
| `pharmacy.activated` | commerce | search, comms | `pharmacy_id` |
| `order.placed` | orders | billing, commerce | `order_id`, `user_id`, `total_centavos` |
| `order.paid` | orders | delivery, comms | `order_id` |
| `order.cancelled` | orders | commerce, billing, comms | `order_id` |
| `order.delivered` | orders | comms | `order_id` |
| `delivery.assigned` | delivery | orders, comms | `delivery_id`, `courier_id`, `order_id` |
| `tracking.updated` | delivery | comms | `delivery_id`, `lat`, `lng` |
| `delivery.completed` | delivery | orders, billing, comms | `delivery_id`, `order_id` |
| `ticket.created` | support | comms | `ticket_id`, `user_id` |
| `ticket.resolved` | support | comms | `ticket_id` |
| `notification.sent` | comms | — | `notification_id`, `user_id`, `canal` |
| `chat.message.created` | comms | — | `room_id`, `message_id` |
| `audit.access` | clinical/files/billing… | audit | `actor_id`, `resource_type`, `resource_id`, `action` |
| `article.updated` | support | search | `article_id` |

## Regras

1. Consumers devem ser **idempotentes** (`event.id`).
2. `version` sobe em breaking change do `data`.
3. `correlation_id` propaga o `X-Request-Id` da requisição origem quando existir.
