# Feature‑Flag API Design

## Transport Protocol

- **gRPC** for internal microservice communication.
- **REST (JSON)** for external/CLI consumers.

Both protocols share the same protobuf definitions for consistency.

## Endpoints

| Method | Path | Description | Auth | RBAC | Request Body | Response Body |
|--------|------|-------------|------|------|---------------|---------------|
| GET | `/flags` | List all flags | JWT | `flag_reader` | none | `FeatureFlag[]` |
| GET | `/flags/{id}` | Retrieve a flag | JWT | `flag_reader` | none | `FeatureFlag` |
| POST | `/flags` | Create a new flag | JWT | `flag_admin` | `FeatureFlagCreate` | `FeatureFlag` |
| PUT | `/flags/{id}` | Update a flag | JWT | `flag_admin` | `FeatureFlagUpdate` | `FeatureFlag` |
| DELETE | `/flags/{id}` | Delete a flag | JWT | `flag_admin` | none | `204 No Content` |
| POST | `/flags/{id}/evaluate` | Evaluate flag for a user | JWT | `flag_reader` | `EvaluationRequest` | `EvaluationResponse` |

### Request/Response Schemas

```proto
message FeatureFlag {
  string id = 1;
  string name = 2;
  string description = 3;
  bool enabled = 4;
  repeated Strategy strategies = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
}

message Strategy {
  string type = 1;
  google.protobuf.Any payload = 2;
}

message EvaluationRequest {
  string user_id = 1;
  repeated string group_ids = 2;
}

message EvaluationResponse {
  bool enabled = 1;
}
```

## Error Handling

- `401 Unauthorized` – missing or invalid JWT.
- `403 Forbidden` – insufficient RBAC role.
- `404 Not Found` – flag id does not exist.
- `422 Unprocessable Entity` – validation errors.
- `500 Internal Server Error` – unexpected errors.

## Versioning

- Use semantic versioning for the API.
- Provide a `/health` endpoint for liveness/readiness checks.

---

**Note**: All gRPC services have corresponding HTTP/JSON endpoints via gRPC‑JSON transcoding.
