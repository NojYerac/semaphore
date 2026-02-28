package data_test

import (
	"context"
	"encoding/json"

	. "github.com/nojyerac/semaphore/data"
	mockdata "github.com/nojyerac/semaphore/mocks/data"
	flagpb "github.com/nojyerac/semaphore/pb/flag"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Data", func() {
	Describe("NewDataEngine", func() {
		It("delegates source and engine methods", func() {
			ctx := context.Background()
			source := &mockdata.MockSource{}
			engine := &mockdata.MockDataEngine{}

			de := NewDataEngine(source, engine)

			expectedFlags := []*FeatureFlag{{ID: "abc", Name: "test-flag"}}
			source.On("GetFlags", mock.Anything).Return(expectedFlags, nil).Once()
			engine.On("EvaluateFlag", mock.Anything, "flag-id", "user-id", []string{"group-id"}).Return(true, nil).Once()

			flags, err := de.GetFlags(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(flags).To(Equal(expectedFlags))

			result, err := de.EvaluateFlag(ctx, "flag-id", "user-id", []string{"group-id"})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())

			source.AssertExpectations(GinkgoT())
			engine.AssertExpectations(GinkgoT())
		})
	})

	Describe("Strategies.Scan", func() {
		It("returns empty strategies when value is nil", func() {
			var strategies Strategies
			err := strategies.Scan(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(strategies).To(HaveLen(0))
		})

		It("returns error for invalid value type", func() {
			var strategies Strategies
			err := strategies.Scan("not-bytes")
			Expect(err).To(MatchError("invalid type for strategies: string"))
		})

		It("returns error for invalid JSON", func() {
			var strategies Strategies
			err := strategies.Scan([]byte(`{"not":"an-array"}`))
			Expect(err).To(MatchError(ContainSubstring("failed to unmarshal strategies")))
		})

		It("scans valid strategies JSON", func() {
			var strategies Strategies
			err := strategies.Scan([]byte(`[{"type":"user_targeting","payload":{"user_ids":["user1"]}}]`))
			Expect(err).ToNot(HaveOccurred())
			Expect(strategies).To(HaveLen(1))
			Expect(strategies[0].Type).To(Equal("user_targeting"))
			Expect(strategies[0].Payload).To(MatchJSON(`{"user_ids":["user1"]}`))
		})
	})

	Describe("Strategy.Scan", func() {
		It("sets zero values when value is nil", func() {
			s := Strategy{Type: "will-be-reset", Payload: json.RawMessage(`{"x":1}`)}
			err := s.Scan(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Type).To(Equal(""))
			Expect(s.Payload).To(BeNil())
		})

		It("returns error for invalid value type", func() {
			var s Strategy
			err := s.Scan(123)
			Expect(err).To(MatchError("invalid type for strategy: int"))
		})

		It("returns error for invalid JSON", func() {
			var s Strategy
			err := s.Scan([]byte(`{"type":`))
			Expect(err).To(MatchError(ContainSubstring("failed to unmarshal strategy")))
		})

		It("scans valid strategy JSON", func() {
			var s Strategy
			err := s.Scan([]byte(`{"type":"group_targeting","payload":{"group_ids":["g1"]}}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Type).To(Equal("group_targeting"))
			Expect(s.Payload).To(MatchJSON(`{"group_ids":["g1"]}`))
		})
	})

	Describe("Strategy.ToProto", func() {
		It("returns nil for nil strategy", func() {
			var s *Strategy
			pb, err := s.ToProto()
			Expect(err).ToNot(HaveOccurred())
			Expect(pb).To(BeNil())
		})

		It("returns error for unknown strategy type", func() {
			s := &Strategy{Type: "unknown", Payload: json.RawMessage(`{}`)}
			pb, err := s.ToProto()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown payload type"))
			Expect(pb).To(BeNil())
		})

		It("converts user targeting strategy", func() {
			s := &Strategy{Type: "user_targeting", Payload: json.RawMessage(`{"user_ids":["u1","u2"]}`)}
			pb, err := s.ToProto()
			Expect(err).ToNot(HaveOccurred())
			Expect(pb.GetType()).To(Equal("user_targeting"))
			Expect(pb.GetUserTargeting().GetUserIds()).To(Equal([]string{"u1", "u2"}))
		})
	})

	Describe("FeatureFlag.ToProto", func() {
		It("converts flag and skips empty strategy types", func() {
			f := &FeatureFlag{
				ID:          "id-1",
				Name:        "my-flag",
				Description: "desc",
				Enabled:     true,
				Strategies: Strategies{
					{Type: "", Payload: json.RawMessage(`{"ignored":true}`)},
					{Type: "percentage_rollout", Payload: json.RawMessage(`{"percentage":42}`)},
				},
			}

			pb, err := f.ToProto()
			Expect(err).ToNot(HaveOccurred())
			Expect(pb.GetId()).To(Equal("id-1"))
			Expect(pb.GetName()).To(Equal("my-flag"))
			Expect(pb.GetDescription()).To(Equal("desc"))
			Expect(pb.GetEnabled()).To(BeTrue())
			Expect(pb.GetStrategies()).To(HaveLen(1))
			Expect(pb.GetStrategies()[0].GetType()).To(Equal("percentage_rollout"))
			Expect(pb.GetStrategies()[0].GetPercentageRollout().GetPercentage()).To(Equal(int32(42)))
		})

		It("returns error when strategy conversion fails", func() {
			f := &FeatureFlag{
				Name:       "my-flag",
				Strategies: Strategies{{Type: "unknown", Payload: json.RawMessage(`{}`)}},
			}

			pb, err := f.ToProto()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown payload type"))
			Expect(pb).To(BeNil())
		})
	})

	Describe("FeatureFlagFromProto", func() {
		It("returns error for nil protobuf flag", func() {
			f, err := FeatureFlagFromProto(nil)
			Expect(err).To(MatchError("nil protobuf flag"))
			Expect(f).To(BeNil())
		})

		It("returns error for unknown strategy payload", func() {
			pb := &flagpb.Flag{
				Name: "test-flag",
				Strategies: []*flagpb.Strategy{
					{Type: "user_targeting"},
				},
			}

			f, err := FeatureFlagFromProto(pb)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown strategy payload type"))
			Expect(f).To(BeNil())
		})

		It("converts protobuf flag with all strategy payloads", func() {
			pb := &flagpb.Flag{
				Id:          "123e4567-e89b-42d3-a456-426614174000",
				Name:        "test-flag",
				Description: "desc",
				Enabled:     true,
				Strategies: []*flagpb.Strategy{
					{
						Type:    "percentage_rollout",
						Payload: &flagpb.Strategy_PercentageRollout{PercentageRollout: &flagpb.PercentageRollout{Percentage: 25}},
					},
					{
						Type:    "user_targeting",
						Payload: &flagpb.Strategy_UserTargeting{UserTargeting: &flagpb.UserTargeting{UserIds: []string{"u1", "u2"}}},
					},
					{
						Type:    "group_targeting",
						Payload: &flagpb.Strategy_GroupTargeting{GroupTargeting: &flagpb.GroupTargeting{GroupIds: []string{"g1"}}},
					},
				},
			}

			f, err := FeatureFlagFromProto(pb)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.ID).To(Equal(pb.Id))
			Expect(f.Name).To(Equal(pb.Name))
			Expect(f.Description).To(Equal(pb.Description))
			Expect(f.Enabled).To(BeTrue())
			Expect(f.Strategies).To(HaveLen(3))
			Expect(f.Strategies[0].Payload).To(MatchJSON(`{"percentage":25}`))
			Expect(f.Strategies[1].Payload).To(MatchJSON(`{"user_ids":["u1","u2"]}`))
			Expect(f.Strategies[2].Payload).To(MatchJSON(`{"group_ids":["g1"]}`))
		})

		It("returns validation error for invalid protobuf flag", func() {
			pb := &flagpb.Flag{Id: "123e4567-e89b-42d3-a456-426614174000"}

			f, err := FeatureFlagFromProto(pb)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Name"))
			Expect(f).To(BeNil())
		})
	})
})
