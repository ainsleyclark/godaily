# Writing a Service

Services should own business logic and depend on interfaces, not implementations.

## service.go

The service.go file should contain the Service type, the New constructor and the interface
compliance only.

### Service Struct

The service struct contains the dependencies required to perform its work. Keep dependencies private
and inject them through the constructor.

```go
type Service struct {
	repo   audience.SubscriberRepository
	issues digest.IssueRepository
	email  email.Sender
}
```

### Constructor

Expose a `New` function that wires dependencies into the service.

```go
func New(
	repo audience.SubscriberRepository,
	issues digest.IssueRepository,
	sender email.Sender,
) *Service {
	return &Service{
		repo:   repo,
		issues: issues,
		email:  sender,
	}
}
```

Constructors should:

- Accept interfaces
- Return the concrete service
- Avoid hidden dependencies

### Interface Compliance

Use a compile-time assertion to ensure the service satisfies its interface. This prevents interface
drift during refactoring. The interface should reference a type in the domain package.

```go
var _ audience.SubscriberService = (*Service)(nil)
```

## Method Receivers

Service methods should generally use pointer receivers.

```go
func (s *Service) Subscribe(ctx context.Context, email string) error {
// ...
}
```

Use:

- `s` as the receiver alias
- Pointer receivers unless immutability is explicitly required

Pointer receivers:

- Avoid copying dependencies
- Keep method sets consistent
- Allow future state additions safely
