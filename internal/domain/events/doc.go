// Package events defines domain event types and the event publisher.
//
// Events are the primary mechanism for loose coupling between domain,
// service, and presentation layers. The event publisher supports
// synchronous guaranteed delivery (QoS 1) for critical events such as
// orders, fills, and risk decisions.
//
// This package must not import providers, controllers, or infrastructure
// code. It depends only on domain/entity.
package events
