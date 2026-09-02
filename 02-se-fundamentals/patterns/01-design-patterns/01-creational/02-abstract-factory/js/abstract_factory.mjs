export class PatternError extends Error { constructor(code){super(code);this.code=code;} }
class AwsFactory { get family(){return "aws";} queue(p){return `sqs:${p}`;} objectStore(p){return `s3:${p}`;} }
class GcpFactory { get family(){return "gcp";} queue(p){return `pubsub:${p}`;} objectStore(p){return `gcs:${p}`;} }
export function evaluate(input) {
  const factories = { aws: AwsFactory, gcp: GcpFactory };
  const resources = (input.providers ?? []).map((provider) => {
    const Factory = factories[provider];
    if (Factory === undefined) throw new PatternError("unsupported_provider");
    const factory = new Factory();
    return { family: factory.family, queue: factory.queue(input.prefix), object_store: factory.objectStore(input.prefix) };
  });
  return { resources };
}
