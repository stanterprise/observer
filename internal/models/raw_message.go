package models

import "time"

// RetainedMessage is a single NATS message captured within a run's retention log.
// Multiple RetainedMessages are embedded in a RawMessagesRunDocument.
type RetainedMessage struct {
	// Subject is the NATS subject on which the message was received.
	Subject string `bson:"subject" json:"subject"`

	// EventType is the decoded event type from the message envelope.
	EventType string `bson:"event_type" json:"eventType"`

	// Payload contains the parsed JSON of the full NATS message (event envelope).
	Payload interface{} `bson:"payload" json:"payload"`

	// Sequence is the JetStream sequence number of the message within the stream.
	Sequence uint64 `bson:"sequence,omitempty" json:"sequence,omitempty"`

	// Stream is the name of the JetStream stream this message was consumed from.
	Stream string `bson:"stream,omitempty" json:"stream,omitempty"`

	// ReceivedAt is the time the message was received and stored.
	ReceivedAt time.Time `bson:"received_at" json:"receivedAt"`
}

// RawMessagesRunDocument groups all retained NATS messages that belong to a single
// test run.  The document _id is the run_id, so every message for a run is
// co-located in one document, making replay and audit straightforward.
type RawMessagesRunDocument struct {
	// RunID is the test run identifier; used as the MongoDB _id.
	RunID string `bson:"_id" json:"runId"`

	// Messages is the ordered list of raw messages received for this run.
	Messages []RetainedMessage `bson:"messages" json:"messages"`

	// CreatedAt is the time the document was first created (first message for the run).
	CreatedAt time.Time `bson:"created_at" json:"createdAt"`

	// UpdatedAt is the time the last message was appended to this document.
	UpdatedAt time.Time `bson:"updated_at" json:"updatedAt"`
}
