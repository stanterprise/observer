// RetainedMessage is a single NATS message captured during a test run.
export interface RetainedMessage {
  subject: string;
  eventType: string;
  // payload is the full event envelope parsed as a JSON object
  payload: unknown;
  sequence?: number;
  stream?: string;
  receivedAt: string;
}

// RawMessagesRunDocument groups all retained messages for a single test run.
export interface RawMessagesRunDocument {
  runId: string;
  messages: RetainedMessage[];
  createdAt: string;
  updatedAt: string;
}
