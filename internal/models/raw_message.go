package models

import "time"

// RawMessageDocument stores the raw NATS message payload as received by the consumer.
// It is used for auditing, debugging, and replay purposes when message retention is enabled.
type RawMessageDocument struct {
	// ID is a unique identifier for this stored message (hex-encoded ObjectID).
	ID string `bson:"_id" json:"id"`

	// Subject is the NATS subject on which the message was received.
	Subject string `bson:"subject" json:"subject"`

	// EventType is the decoded event type from the message envelope.
	EventType string `bson:"event_type" json:"eventType"`

	// Payload contains the raw JSON bytes of the full NATS message (event envelope).
	Payload []byte `bson:"payload" json:"payload"`

	// Sequence is the JetStream sequence number of the message within the stream.
	Sequence uint64 `bson:"sequence,omitempty" json:"sequence,omitempty"`

	// Stream is the name of the JetStream stream this message was consumed from.
	Stream string `bson:"stream,omitempty" json:"stream,omitempty"`

	// ReceivedAt is the time the message was received and stored.
	ReceivedAt time.Time `bson:"received_at" json:"receivedAt"`
}
