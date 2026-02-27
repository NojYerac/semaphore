# Feature‑Flag Client SDK

## Package Overview

```
semaphore/sdk/
├── client.go       // public API
├── auth.go         // JWT handling
└── util.go         // helpers for strategy evaluation
```

## Usage Example

```go
import "semaphore/sdk"

client, err := sdk.NewClient("https://flags.example.com", "<JWT>")
if err != nil { log.Fatal(err) }

flag, err := client.GetFlag(ctx, "new-feature")
if err != nil { log.Fatal(err) }

enabled, err := client.Evaluate(ctx, flag.ID, sdk.EvaluationRequest{UserID: "user-123"})
if err != nil { log.Fatal(err) }
fmt.Println("Feature enabled?", enabled)
```

## API Methods

- `GetFlag(ctx, id)` – retrieves a flag.
- `ListFlags(ctx)` – lists all flags.
- `CreateFlag(ctx, flag)` – creates a new flag.
- `UpdateFlag(ctx, flag)` – updates a flag.
- `DeleteFlag(ctx, id)` – deletes a flag.
- `Evaluate(ctx, id, req)` – evaluates a flag for a context.

All methods return Go structs that mirror the protobuf definitions.

## Authentication

The SDK accepts a JWT token that is sent in the `Authorization: Bearer <token>` header for every request.

## Tests

Place tests in `sdk/` and use the `httptest` package to mock the server.

---

**Next**: Write integration tests and CI configuration.
