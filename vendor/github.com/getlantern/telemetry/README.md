# telemetry
Common library for any Go code that wants to interface with telemetry.

`telemetry` **expects everything to be configured via default otel environment variables to maximize portability and flexibility**.

To modify the sample rate and sampling strategy, for example, you can use:

```
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.001
```

See https://opentelemetry-python.readthedocs.io/en/latest/sdk/trace.sampling.html

## Setup

`telemetry` is designed to minimize the boilerplate needed to get up and running quickly. To enable it, you simply add the following:

```go
ctx := context.Background()
closeFunc := telemetry.EnableOTELTracing(ctx)
defer func() { _ = closeFunc(ctx) }()
```

From that point on, tracing is configured for you, and you can use it as normal. For example, you can run:

```go
tracer := otel.Tracer("my-tracer")
```

## Sampling 
This library also contains convenience functions for forcing certain traces to be sampled. For example, to always sample HTTP requests with a specific HTTP header and value, you can run:

```go
telemetry.AlwaysSampleHeaderHandler("name", "value", otelhttp.NewHandler(...)))
```

If the value is "*", then it will always sample requests that have the header set to any value.

`telemetry` includes this ability to force traces to be sampled more generally, though. If you want to force a trace to be sampled, you can call the following to add forced sampling to its context:

```go
ctx := telemetry.AlwaysSample(r.Context())
```

That should be done for the **root span**.
