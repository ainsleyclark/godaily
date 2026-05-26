# TODO

## App & General

Each domain should define their services and types in the domain package, the service should
implement the methods.

I want to pass around this service struct in the app. We should strive for each domain type to have
a service. Currently, I think social is lacking one, which is why we have to so socialsvc.Service.

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
