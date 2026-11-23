package metrics

import "github.com/zeromicro/go-zero/core/metric"

var (
	WSActiveConnections = metric.NewGaugeVec(
		&metric.GaugeVecOpts{
			Namespace: "easy_chat",
			Subsystem: "ws",
			Name:      "active_connections",
			Help:      "Current active WebSocket connections.",
			Labels:    []string{"node"},
		},
	)

	WSMessagesTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Namespace: "easy_chat",
			Subsystem: "ws",
			Name:      "messages_total",
			Help:      "WebSocket messages partitioned by type and result.",
			Labels:    []string{"node", "type", "result"},
		},
	)

	MessagePersistedTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Namespace: "easy_chat",
			Subsystem: "message",
			Name:      "persisted_total",
			Help:      "Messages processed by persistence outcome.",
			Labels:    []string{"result"},
		},
	)

	MessageRejectedTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Namespace: "easy_chat",
			Subsystem: "message",
			Name:      "rejected_total",
			Help:      "Message commands rejected by reason.",
			Labels:    []string{"reason"},
		},
	)

	MessagePushTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Namespace: "easy_chat",
			Subsystem: "message",
			Name:      "push_total",
			Help:      "Message push attempts by result.",
			Labels:    []string{"result"},
		},
	)

	ReceiptProcessedTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Namespace: "easy_chat",
			Subsystem: "receipt",
			Name:      "processed_total",
			Help:      "Receipt commands processed by result.",
			Labels:    []string{"type", "result"},
		},
	)

	DeadLettersTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Namespace: "easy_chat",
			Subsystem: "kafka",
			Name:      "dead_letters_total",
			Help:      "Kafka records written to the dead-letter topic.",
			Labels:    []string{"source_topic", "consumer_group"},
		},
	)

	OutboxPending = metric.NewGaugeVec(
		&metric.GaugeVecOpts{
			Namespace: "easy_chat",
			Subsystem: "outbox",
			Name:      "pending",
			Help:      "Current pending outbox events.",
			Labels:    []string{"kind"},
		},
	)
)
