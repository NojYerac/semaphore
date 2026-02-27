# Feature‑Flag Engine Core Library

## Package Layout

```
semaphore/engine/
├── flag.go        // FeatureFlag, Strategy structs
├── evaluator.go   // Evaluation logic
├── repository.go  // Persistence abstraction
└── audit.go       // Audit event handling
```

## Evaluation Algorithm

```go
func (e *Evaluator) Evaluate(flag FeatureFlag, ctx EvaluationContext) bool {
    if !flag.Enabled {
        return false
    }
    for _, s := range flag.Strategies {
        switch s.Type {
        case "percentage":
            if evalPercentage(s.Payload, ctx) { return true }
        case "user":
            if evalUser(s.Payload, ctx) { return true }
        case "group":
            if evalGroup(s.Payload, ctx) { return true }
        }
    }
    return false
}
```

### Strategy Evaluators

- **Percentage** – deterministic hash of user id mod 100 compared to payload percentage.
- **User** – membership check in `user_ids` array.
- **Group** – membership check in `group_ids` array.

## Repository Interface

```go
type FlagRepository interface {
    Create(ctx context.Context, f *FeatureFlag) error
    GetByID(ctx context.Context, id string) (*FeatureFlag, error)
    List(ctx context.Context) ([]*FeatureFlag, error)
    Update(ctx context.Context, f *FeatureFlag) error
    Delete(ctx context.Context, id string) error
}
```

## Audit Events

```go
type AuditEvent struct {
    ID          string
    FlagID      *string
    ActorID     string
    Action      string
    Details     json.RawMessage
    OccurredAt  time.Time
}
```

## Unit Tests

Place tests in `engine/` with the naming convention `*_test.go`.

---

**Next**: Create the API server skeleton and middleware.
