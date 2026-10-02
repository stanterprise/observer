# Reporter Integration

## Playwright Reporter

The available reporter is the published [`@stanterprise/playwright-reporter`](https://www.npmjs.com/package/@stanterprise/playwright-reporter). It uses a gRPC client generated from the shared protobuf definitions.

It reports event families for:

- RunStarted / RunFinished
- TestStarted / TestFinished
- StepStarted / StepFinished
- AttachmentAdded

Current reporter configuration uses `grpcAddress`:

```typescript
reporter: [
  ["@stanterprise/playwright-reporter", { grpcAddress: "localhost:50051" }],
];
```

The reporter supports retries with exponential backoff, attachment handling, sharding metadata, and custom run metadata. See the [public reporter guide](https://observer.stanterprise.com/docs/integrations/playwright-reporter/) for the full current option list.

## Event Delivery

- Events are sent over gRPC to ingestion and published to NATS JetStream.
- The reporter retries transient gRPC failures with exponential backoff.
- Attachments are handled by the reporter and stored through the configured local or S3-compatible storage driver.
- Earlier design notes specified 64–256 KB attachment chunks and a local spillover queue when the broker is offline. Those details are not guaranteed by the current reporter contract.

## Legacy Configuration

Early examples used `OBSERVER_ENDPOINT`, `OBSERVER_TOKEN`, or an `endpoint` option. These are not current reporter settings; use `grpcAddress` and the `STANTERPRISE_*` environment variables documented in the public guide.

## Other Frameworks

- Pytest and Mocha reporters are in progress.
- JUnit and Jest reporter integrations remain future work.
- Do not treat a shared event protocol as proof that a framework reporter is already available.
