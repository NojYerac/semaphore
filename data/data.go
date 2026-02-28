package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/nojyerac/semaphore/pb/flag"
)

type DataEngine interface {
	Engine
	Source
}

type dataEngine struct {
	Engine
	Source
}

var _ DataEngine = (*dataEngine)(nil)

func NewDataEngine(source Source, engine Engine) DataEngine {
	return &dataEngine{
		Engine: engine,
		Source: source,
	}
}

type Engine interface {
	EvaluateFlag(ctx context.Context, flagID, userID string, groupIDs []string) (bool, error)
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
	ID          string     `json:"id" db:"id" validate:"omitempty,uuid4"`
	Name        string     `json:"name" db:"name" validate:"required"`
	Description string     `json:"description,omitempty" db:"description"`
	Enabled     bool       `json:"enabled" db:"enabled"`
	Strategies  Strategies `json:"strategies,omitempty" db:"strategies" validate:"dive"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
}

func (f *FeatureFlag) ToProto() (*flag.Flag, error) {
	pb := &flag.Flag{
		Id:          f.ID,
		Name:        f.Name,
		Description: f.Description,
		Enabled:     f.Enabled,
		Strategies:  make([]*flag.Strategy, 0, len(f.Strategies)),
	}
	for _, s := range f.Strategies {
		if s.Type == "" {
			continue
		}
		pbStrat, err := s.ToProto()
		if err != nil {
			return nil, err
		}
		pb.Strategies = append(pb.Strategies, pbStrat)
	}
	return pb, nil
}

type Strategies []Strategy

var _ sql.Scanner = (*Strategies)(nil)

func (f *Strategies) Scan(value interface{}) error {
	*f = make(Strategies, 0)
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("invalid type for strategies: %T", value)
	}
	if err := json.Unmarshal(b, f); err != nil {
		return fmt.Errorf("failed to unmarshal strategies: %w", err)
	}
	return nil
}

// Strategy defines an evaluation rule.
// Type is one of "percentage_rollout", "user_targeting", or "group_targeting".
// Payload is a raw JSON blob containing the strategy specific data.
// The engine unmarshals it based on the Type value.
type Strategy struct {
	Type    string          `json:"type" db:"type" validate:"required,oneof=percentage_rollout user_targeting group_targeting"` //nolint:lll // param tag
	Payload json.RawMessage `json:"payload" db:"payload"`
}

var _ sql.Scanner = (*Strategy)(nil)

func (s *Strategy) Scan(value interface{}) error {
	*s = Strategy{}
	if value == nil {
		s.Type = ""
		s.Payload = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("invalid type for strategy: %T", value)
	}
	if err := json.Unmarshal(b, s); err != nil {
		return fmt.Errorf("failed to unmarshal strategy: %w", err)
	}
	return nil
}

func (s *Strategy) ToProto() (*flag.Strategy, error) {
	if s == nil {
		return nil, nil
	}
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
	case "percentage_rollout":
		v, ok := data["percentage"]
		if !ok {
			return nil
		}
		f, ok := v.(float64)
		if !ok {
			return nil
		}
		return &flag.Strategy_PercentageRollout{
			PercentageRollout: &flag.PercentageRollout{
				Percentage: int32(f),
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

var validate = validator.New()

func FeatureFlagFromProto(pb *flag.Flag) (*FeatureFlag, error) {
	if pb == nil {
		return nil, fmt.Errorf("nil protobuf flag")
	}
	f := &FeatureFlag{
		ID:          pb.Id,
		Name:        pb.Name,
		Description: pb.Description,
		Enabled:     pb.Enabled,
		Strategies:  make(Strategies, len(pb.Strategies)),
	}
	for i, s := range pb.Strategies {
		f.Strategies[i] = Strategy{
			Type: s.GetType(),
		}
		switch p := s.GetPayload().(type) {
		case *flag.Strategy_PercentageRollout:
			f.Strategies[i].Payload, _ = json.Marshal(map[string]interface{}{
				"percentage": p.PercentageRollout.GetPercentage(),
			})
		case *flag.Strategy_UserTargeting:
			f.Strategies[i].Payload, _ = json.Marshal(map[string]interface{}{
				"user_ids": p.UserTargeting.GetUserIds(),
			})
		case *flag.Strategy_GroupTargeting:
			f.Strategies[i].Payload, _ = json.Marshal(map[string]interface{}{
				"group_ids": p.GroupTargeting.GetGroupIds(),
			})
		default:
			return nil, fmt.Errorf("unknown strategy payload type: %T", s.GetPayload())
		}
	}
	if err := validate.Struct(f); err != nil {
		return nil, fmt.Errorf("invalid feature flag: %w", err)
	}
	return f, nil
}

// AuditLog represents a log entry for flag operations.
type AuditLog struct {
	ID        string          `json:"id" db:"id" validate:"required,uuid4"`
	FlagID    string          `json:"flagID" db:"flag_id" validate:"required,uuid4"`
	Action    string          `json:"action" db:"action" validate:"required,oneof=create update delete"` //nolint:lll // param tag
	Timestamp time.Time       `json:"timestamp" db:"timestamp" validate:"required"`
	User      string          `json:"userID" db:"user_id" validate:"required,uuid4"`
	Details   json.RawMessage `json:"details" db:"details"` // JSON string with operation details
}
