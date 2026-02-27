package engine

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nojyerac/go-lib/log"
	"github.com/nojyerac/semaphore/data"
	"github.com/sirupsen/logrus"
)

var _ data.Engine = (*Engine)(nil)

type Engine struct {
	source data.Source
}

type Option func(*Engine)

func NewEngine(source data.Source, opts ...Option) *Engine {
	engine := &Engine{source: source}
	for _, opt := range opts {
		opt(engine)
	}
	return engine
}

type payload struct {
	UserIDs    []string `json:"user_ids,omitempty"`
	GroupIDs   []string `json:"group_ids,omitempty"`
	Percentage int32    `json:"percentage,omitempty"`
}

func (e *Engine) EvaluateFlag(ctx context.Context, flagID, userID string, groupIDs []string) (bool, error) {
	logger := log.FromContext(ctx).WithFields(logrus.Fields{"flagID": flagID, "userID": userID, "groupIDs": groupIDs})
	flag, err := e.source.GetFlagByID(ctx, flagID)
	if err != nil {
		return false, err
	}
	logger.Debug("retrieved flag", "flag", flag, "err", err)
	if !flag.Enabled {
		return false, nil
	}
	logger.Debug("flag enabled")
	if len(flag.Strategies) == 0 {
		return true, nil
	}
	logger.Debugf("evaluating %d strategies", len(flag.Strategies))
	for _, strategy := range flag.Strategies {
		pl := &payload{}
		if err := json.Unmarshal(strategy.Payload, pl); err != nil {
			return false, err
		}
		logger.Debug("evaluating strategy", "type", strategy.Type, "payload", pl)
		switch strategy.Type {
		case "user_targeting":
			if evaluateUserTargetingStrategy(pl, userID) {
				return true, nil
			}
		case "group_targeting":
			if evaluateGroupTargetingStrategy(pl, groupIDs) {
				return true, nil
			}
		case "percentage":
			enabled, err := evaluatePercentageStrategy(flagID, userID, pl)
			if err != nil {
				return false, err
			}
			if enabled {
				return true, nil
			}
		default:
			return false, fmt.Errorf("unknown strategy type: %s", strategy.Type)
		}
	}
	logger.Debug("no strategies matched")
	return false, nil
}

func evaluateUserTargetingStrategy(pl *payload, userID string) bool {
	for _, id := range pl.UserIDs {
		if id == userID {
			return true
		}
	}

	return false
}

func evaluateGroupTargetingStrategy(pl *payload, groupIDs []string) bool {
	for _, id := range pl.GroupIDs {
		for _, groupID := range groupIDs {
			if id == groupID {
				return true
			}
		}
	}

	return false
}

func evaluatePercentageStrategy(flagID, userID string, pl *payload) (bool, error) {
	// UserID is a uuid. We can use it to get a consistent value for the user.
	f, err := uuid.Parse(flagID)
	if err != nil {
		return false, err
	}
	u, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	// XOR the two UUIDs to get a consistent value for the user and flag combination.
	fxoru := make([]byte, 4)
	for i := 0; i < 4; i++ {
		fxoru[i] = f[i] ^ u[i]
	}
	// Get a value between 0 and 99 for the flag & user combination.
	percentile := int32(binary.BigEndian.Uint32(fxoru) % 100) //nolint:gosec // 0-99 can't overflow int32
	// If the value is less than the percentage rollout, the flag is enabled for the user.
	return percentile < pl.Percentage, nil
}
