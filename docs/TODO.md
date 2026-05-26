# TODO

## App & General

Each domain should define their services and types in the domain package, the service should
implement the methods.

I want to pass around this service struct in the app. We should strive for each domain type to have
a service. Currently, I think social is lacking one, which is why we have to so socialsvc.Service.
As well as metrics/engagement.

```go
type Service struct {
Digest      digest.Service
Subscribers audience.SubscriberService
Social      *socialsvc.Service
}
```

## Digest Service

pkg/services/digest

- Some of the stuff in helpers_test.go we don't need, such as mockSlack, there are already
  generated mock stubs for this, as well asc mockEmail? It should all be in the mocks package, let
  me know if not.
- I don't want a separate recap service, it should be part of the main digest one.

## Social Service

pkg/services/social

- I don't like that buildRotationCandidates is in app package, the social package should contain
  this and not expose WithCandidates or HasPosters, ideally HasPosters should be private and only
  evlauted before posting. As for the WithCandidates function, the social service should take env
  config and bootstrap all the platforms.
- This buildStatFetchers should also be social service, not in the app layer.

## Metrics Domain

Should this not be MetricsService? We currently have MetricsReporter and SocialMetricRepository
which seems confusing.

```// MetricsReporter produces higher-level engagement reports composed from
// MetricsRepository queries. It is the interface API handlers depend on so
// they can be tested without orchestrating every underlying query.
type MetricsReporter interface {
	// Roundup gathers the last seven days of metrics (with a week-over-week
	// comparison) and posts a formatted summary to the configured Slack channel.
	Roundup(ctx context.Context) error
}```
