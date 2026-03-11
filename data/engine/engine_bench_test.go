package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/nojyerac/semaphore/data"
	. "github.com/nojyerac/semaphore/data/engine"
	mockdata "github.com/nojyerac/semaphore/mocks/data"
	"github.com/stretchr/testify/mock"
)

// BenchmarkEvaluateFlag_SingleFlag benchmarks single flag evaluation with different strategies
func BenchmarkEvaluateFlag_SingleFlag(b *testing.B) {
	testCases := []struct {
		name     string
		strategy data.Strategy
	}{
		{
			name: "PercentageRollout",
			strategy: data.Strategy{
				Type:    "percentage_rollout",
				Payload: json.RawMessage(`{"percentage": 50}`),
			},
		},
		{
			name: "UserTargeting",
			strategy: data.Strategy{
				Type:    "user_targeting",
				Payload: mustMarshal(map[string]interface{}{"user_ids": []string{uuid.New().String()}}),
			},
		},
		{
			name: "GroupTargeting",
			strategy: data.Strategy{
				Type:    "group_targeting",
				Payload: mustMarshal(map[string]interface{}{"group_ids": []string{uuid.New().String()}}),
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			source := &mockdata.MockSource{}
			engine := NewEngine(source)
			ctx := context.Background()

			flagID := uuid.New().String()
			userID := uuid.New().String()
			groupIDs := []string{uuid.New().String()}

			flag := &data.FeatureFlag{
				ID:         flagID,
				Name:       "test-flag",
				Enabled:    true,
				Strategies: []data.Strategy{tc.strategy},
			}

			source.On("GetFlagByID", mock.Anything, flagID).Return(flag, nil)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = engine.EvaluateFlag(ctx, flagID, userID, groupIDs)
			}
		})
	}
}

// BenchmarkEvaluateFlag_NoStrategies benchmarks flag evaluation with no strategies (fast path)
func BenchmarkEvaluateFlag_NoStrategies(b *testing.B) {
	source := &mockdata.MockSource{}
	engine := NewEngine(source)
	ctx := context.Background()

	flagID := uuid.New().String()
	userID := uuid.New().String()
	groupIDs := []string{uuid.New().String()}

	flag := &data.FeatureFlag{
		ID:         flagID,
		Name:       "test-flag",
		Enabled:    true,
		Strategies: []data.Strategy{},
	}

	source.On("GetFlagByID", mock.Anything, flagID).Return(flag, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.EvaluateFlag(ctx, flagID, userID, groupIDs)
	}
}

// BenchmarkEvaluateFlag_MultipleStrategies benchmarks flag evaluation with multiple strategies
func BenchmarkEvaluateFlag_MultipleStrategies(b *testing.B) {
	strategies := []int{1, 3, 5, 10}

	for _, count := range strategies {
		b.Run(fmt.Sprintf("%dStrategies", count), func(b *testing.B) {
			source := &mockdata.MockSource{}
			engine := NewEngine(source)
			ctx := context.Background()

			flagID := uuid.New().String()
			userID := uuid.New().String()
			groupIDs := []string{uuid.New().String()}

			strats := make([]data.Strategy, count)
			for i := 0; i < count; i++ {
				strats[i] = data.Strategy{
					Type:    "percentage_rollout",
					Payload: json.RawMessage(`{"percentage": 10}`), // Low percentage so we test all strategies
				}
			}

			flag := &data.FeatureFlag{
				ID:         flagID,
				Name:       "test-flag",
				Enabled:    true,
				Strategies: strats,
			}

			source.On("GetFlagByID", mock.Anything, flagID).Return(flag, nil)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = engine.EvaluateFlag(ctx, flagID, userID, groupIDs)
			}
		})
	}
}

// BenchmarkEvaluateFlag_Concurrent benchmarks concurrent flag evaluation
func BenchmarkEvaluateFlag_Concurrent(b *testing.B) {
	concurrency := []int{10, 100, 1000}

	for _, workers := range concurrency {
		b.Run(fmt.Sprintf("%dGoroutines", workers), func(b *testing.B) {
			source := &mockdata.MockSource{}
			engine := NewEngine(source)
			ctx := context.Background()

			flagID := uuid.New().String()
			flag := &data.FeatureFlag{
				ID:      flagID,
				Name:    "test-flag",
				Enabled: true,
				Strategies: []data.Strategy{
					{
						Type:    "percentage_rollout",
						Payload: json.RawMessage(`{"percentage": 50}`),
					},
				},
			}

			source.On("GetFlagByID", mock.Anything, flagID).Return(flag, nil)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				userID := uuid.New().String()
				groupIDs := []string{uuid.New().String()}

				for pb.Next() {
					_, _ = engine.EvaluateFlag(ctx, flagID, userID, groupIDs)
				}
			})
		})
	}
}

// BenchmarkEvaluateFlag_VaryingFlagCounts benchmarks evaluation with different numbers of flags
// This simulates a realistic scenario where the source might cache multiple flags
func BenchmarkEvaluateFlag_VaryingFlagCounts(b *testing.B) {
	flagCounts := []int{10, 100, 1000}

	for _, count := range flagCounts {
		b.Run(fmt.Sprintf("%dFlags", count), func(b *testing.B) {
			source := &mockdata.MockSource{}
			engine := NewEngine(source)
			ctx := context.Background()

			// Create multiple flags
			flags := make([]*data.FeatureFlag, count)
			for i := 0; i < count; i++ {
				flagID := uuid.New().String()
				flags[i] = &data.FeatureFlag{
					ID:      flagID,
					Name:    fmt.Sprintf("test-flag-%d", i),
					Enabled: true,
					Strategies: []data.Strategy{
						{
							Type:    "percentage_rollout",
							Payload: json.RawMessage(`{"percentage": 50}`),
						},
					},
				}
				source.On("GetFlagByID", mock.Anything, flagID).Return(flags[i], nil)
			}

			b.ResetTimer()
			// Rotate through flags to simulate real-world usage
			for i := 0; i < b.N; i++ {
				flag := flags[i%count]
				userID := uuid.New().String()
				groupIDs := []string{uuid.New().String()}
				_, _ = engine.EvaluateFlag(ctx, flag.ID, userID, groupIDs)
			}
		})
	}
}

// BenchmarkEvaluateFlag_ConcurrentVaryingFlags combines concurrent evaluation with multiple flags
func BenchmarkEvaluateFlag_ConcurrentVaryingFlags(b *testing.B) {
	source := &mockdata.MockSource{}
	engine := NewEngine(source)
	ctx := context.Background()

	// Create 100 flags
	flagCount := 100
	flags := make([]*data.FeatureFlag, flagCount)
	for i := 0; i < flagCount; i++ {
		flagID := uuid.New().String()
		flags[i] = &data.FeatureFlag{
			ID:      flagID,
			Name:    fmt.Sprintf("test-flag-%d", i),
			Enabled: true,
			Strategies: []data.Strategy{
				{
					Type:    "percentage_rollout",
					Payload: json.RawMessage(`{"percentage": 50}`),
				},
			},
		}
		source.On("GetFlagByID", mock.Anything, flagID).Return(flags[i], nil)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		userID := uuid.New().String()
		groupIDs := []string{uuid.New().String()}
		i := 0

		for pb.Next() {
			flag := flags[i%flagCount]
			_, _ = engine.EvaluateFlag(ctx, flag.ID, userID, groupIDs)
			i++
		}
	})
}

// BenchmarkEvaluateFlag_RealWorldScenario benchmarks a realistic production scenario:
// - 1000 goroutines (simulating high concurrency)
// - 100 different flags
// - Mix of strategy types
func BenchmarkEvaluateFlag_RealWorldScenario(b *testing.B) {
	source := &mockdata.MockSource{}
	engine := NewEngine(source)
	ctx := context.Background()

	// Create 100 flags with mixed strategies
	flagCount := 100
	flags := make([]*data.FeatureFlag, flagCount)
	strategyTypes := []struct {
		typ     string
		payload json.RawMessage
	}{
		{"percentage_rollout", json.RawMessage(`{"percentage": 50}`)},
		{"user_targeting", mustMarshal(map[string]interface{}{"user_ids": []string{uuid.New().String()}})},
		{"group_targeting", mustMarshal(map[string]interface{}{"group_ids": []string{uuid.New().String()}})},
	}

	for i := 0; i < flagCount; i++ {
		flagID := uuid.New().String()
		stratType := strategyTypes[i%len(strategyTypes)]
		flags[i] = &data.FeatureFlag{
			ID:      flagID,
			Name:    fmt.Sprintf("test-flag-%d", i),
			Enabled: true,
			Strategies: []data.Strategy{
				{
					Type:    stratType.typ,
					Payload: stratType.payload,
				},
			},
		}
		source.On("GetFlagByID", mock.Anything, flagID).Return(flags[i], nil)
	}

	b.ResetTimer()
	b.SetParallelism(1000) // Simulate 1000 goroutines
	b.RunParallel(func(pb *testing.PB) {
		userID := uuid.New().String()
		groupIDs := []string{uuid.New().String()}
		i := 0

		for pb.Next() {
			flag := flags[i%flagCount]
			_, _ = engine.EvaluateFlag(ctx, flag.ID, userID, groupIDs)
			i++
		}
	})
}

// Helper function to marshal JSON without error handling in test setup
func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
