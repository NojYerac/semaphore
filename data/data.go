package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nojyerac/semaphore/pb/flag"
)

type Engine interface {
	EvaluateFlag(ctx context.Context, flagName string, userID string, groupIDs []string) (bool, error)
}
type Source interface {
	GetFlags(ctx context.Context) ([]*FeatureFlag, error)
	GetFlagByID(ctx context.Context, id string) (*FeatureFlag, error)
	CreateFlag(ctx context.Context, flag *FeatureFlag) (string, error)
	UpdateFlag(ctx context.Context, flag *FeatureFlag) error
	DeleteFlag(ctx context.Context, id string) error
}

// FeatureFlag represents a flag definition.
// It mirrors the protobuf and API structures.
//
//  * ID          – UUID primary key.
//  * Name        – Unique name used by clients.
//  * Description – Optional human‑readable description.
//  * Enabled     – Global enabled flag.
//  * Strategies  – Evaluation rules.
//  * CreatedAt, UpdatedAt – Timestamps.
//
// The structs are deliberately simple so they can be marshaled to JSON or protobuf directly.
//
// The flag is immutable after creation; updates replace the entire struct.

// FeatureFlag defines the shape of a flag.
type FeatureFlag struct {
	ID          string     `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	Enabled     bool       `json:"enabled" db:"enabled"`
	Strategies  []Strategy `json:"strategies" db:"strategies"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
}

func (f *FeatureFlag) ToProto() (*flag.Flag, error) {
	pb := &flag.Flag{
		Id:          f.ID,
		Name:        f.Name,
		Description: f.Description,
		Enabled:     f.Enabled,
		Strategies:  make([]*flag.Strategy, len(f.Strategies)),
	}
	var err error
	for i, s := range f.Strategies {
		pb.Strategies[i], err = s.ToProto()
		if err != nil {
			return nil, err
		}
	}
	return pb, nil
}

// Strategy defines an evaluation rule.
// Type is one of "percentage", "user", or "group".
// Payload is a raw JSON blob containing the strategy specific data.
// The engine unmarshals it based on the type.
type Strategy struct {
	Type    string          `json:"type" db:"type"`
	Payload json.RawMessage `json:"payload" db:"payload"`
}

func (s *Strategy) ToProto() (*flag.Strategy, error) {
	payload := payloadToProto(s.Type, s.Payload)
	pb := &flag.Strategy{
		Type: s.Type,
	}
	switch p := payload.(type) {
	case *flag.Strategy_PercentageRollout:
		pb.Payload = p
	case *flag.Strategy_UserTargeting:
		pb.Payload = p
	case *flag.Strategy_GroupTargeting:
		pb.Payload = p
	default:
		return nil, fmt.Errorf("unknown payload type: %T", payload)
	}
	return pb, nil
}

func payloadToProto(strategyType string, payload json.RawMessage) interface{} {
	data := make(map[string]interface{}, 1)
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil
	}
	switch strategyType {
	case "percentage":
		return &flag.Strategy_PercentageRollout{
			PercentageRollout: &flag.PercentageRollout{
				Percentage: int32(data["percentage"].(float64)),
			},
		}
	case "user_targeting":
		var is []interface{}
		var ok bool
		if is, ok = data["user_ids"].([]interface{}); !ok {
			return nil
		}
		ut := &flag.Strategy_UserTargeting{
			UserTargeting: &flag.UserTargeting{
				UserIds: make([]string, len(is)),
			},
		}
		for i, v := range is {
			if s, ok := v.(string); ok {
				ut.UserTargeting.UserIds[i] = s
			} else {
				return nil
			}
		}
		return ut
	case "group_targeting":
		var is []interface{}
		var ok bool
		if is, ok = data["group_ids"].([]interface{}); !ok {
			return nil
		}
		gt := &flag.Strategy_GroupTargeting{
			GroupTargeting: &flag.GroupTargeting{
				GroupIds: make([]string, len(is)),
			},
		}
		for i, v := range is {
			if s, ok := v.(string); ok {
				gt.GroupTargeting.GroupIds[i] = s
			} else {
				return nil
			}
		}
		return gt
	default:
		return nil
	}
}

// AuditLog represents a log entry for flag operations.
type AuditLog struct {
	ID        string    `json:"id" db:"id"`
	FlagID    string    `json:"flagID" db:"flag_id"`
	Action    string    `json:"action" db:"action"` // "create", "update", "delete"
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	User      string    `json:"user" db:"user"`
	Details   string    `json:"details" db:"details"` // JSON string with operation details
}
