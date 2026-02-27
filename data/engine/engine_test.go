package engine_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nojyerac/go-lib/log"
	"github.com/nojyerac/semaphore/data"
	. "github.com/nojyerac/semaphore/data/engine"
	mockdata "github.com/nojyerac/semaphore/mocks/data"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var (
	flagID   = uuid.New().String()
	userID   = uuid.New().String()
	groupIDs = []string{uuid.New().String()}
)
var _ = Describe("Engine", func() {
	var (
		engine *Engine
		source *mockdata.MockSource
	)
	BeforeEach(func() {
		source = &mockdata.MockSource{}
		engine = NewEngine(source)
		Expect(engine).ToNot(BeNil())
	})
	Context("EvaluateFlag", func() {
		var (
			ctx              context.Context
			flag             *data.FeatureFlag
			err, expectedErr error
			result           bool
		)
		BeforeEach(func() {
			l := log.NewLogger(&log.Configuration{LogLevel: "debug", HumanFrendly: true}, log.WithOutput(GinkgoWriter))
			ctx = log.WithLogger(context.Background(), l)
			flag = &data.FeatureFlag{
				Name: "test-flag",
				ID:   flagID,
			}
		})
		JustBeforeEach(func() {
			source.On("GetFlagByID", mock.Anything, flagID).Return(flag, err)
			result, expectedErr = engine.EvaluateFlag(ctx, flagID, userID, groupIDs)
		})
		AfterEach(func() {
			source.AssertExpectations(GinkgoT())
		})
		AssertResultTrue := func() {
			GinkgoHelper()
			It("returns true", func() {
				Expect(result).To(BeTrue())
				Expect(expectedErr).ToNot(HaveOccurred())
			})
		}
		AssertResultFalse := func() {
			GinkgoHelper()
			It("returns false", func() {
				Expect(result).To(BeFalse())
				Expect(expectedErr).ToNot(HaveOccurred())
			})
		}
		AssertError := func(msg string) {
			GinkgoHelper()
			It("returns an error", func() {
				Expect(result).To(BeFalse())
				Expect(expectedErr).To(MatchError(msg))
			})
		}
		Context("uuids are valid", func() {
			Context("flag is not found", func() {
				// TODO: check this behavior. Is an error returned by the data layer?
				BeforeEach(func() {
					flag = nil
					err = fmt.Errorf("flag not found")
				})
				AssertError("flag not found")
			})
			Context("flag is disabled", func() {
				BeforeEach(func() {
					flag.Enabled = false
					err = nil
				})
				AssertResultFalse()
			})
			Context("flag is enabled with no strategies", func() {
				BeforeEach(func() {
					flag.Enabled = true
					err = nil
					flag.Strategies = []data.Strategy{}
				})
				AssertResultTrue()
			})
			Context("flag is enabled with invalid strategy payload", func() {
				BeforeEach(func() {
					flag.Enabled = true
					err = nil
					flag.Strategies = []data.Strategy{
						{
							Type:    "user_targeting",
							Payload: []byte(`invalid json`),
						},
					}
				})
				AssertError("invalid character 'i' looking for beginning of value")
			})
			Context("flag is enabled with unknown strategy type", func() {
				BeforeEach(func() {
					flag.Enabled = true
					err = nil
					flag.Strategies = []data.Strategy{
						{
							Type:    "unknown",
							Payload: []byte(`{}`),
						},
					}
				})
				AssertError("unknown strategy type: unknown")
			})
			Context("flag is enabled with user_targeting", func() {
				BeforeEach(func() {
					flag.Enabled = true
					err = nil
					flag.Strategies = []data.Strategy{
						{
							Type:    "user_targeting",
							Payload: []byte(fmt.Sprintf(`{"user_ids": [%q]}`, userID)),
						},
					}
				})
				When("user is targeted", func() {
					AssertResultTrue()
				})
				When("user is not targeted", func() {
					BeforeEach(func() {
						flag.Strategies[0].Payload = []byte(fmt.Sprintf(`{"user_ids": [%q]}`, uuid.New().String()))
					})
					AssertResultFalse()
				})
			})
			Context("flag is enabled with group_targeting", func() {
				BeforeEach(func() {
					flag.Enabled = true
					err = nil
					flag.Strategies = []data.Strategy{
						{
							Type:    "group_targeting",
							Payload: []byte(fmt.Sprintf(`{"group_ids": [%q]}`, groupIDs[0])),
						},
					}
				})
				When("group is targeted", func() {
					AssertResultTrue()
				})
				When("group is not targeted", func() {
					BeforeEach(func() {
						flag.Strategies[0].Payload = []byte(fmt.Sprintf(`{"group_ids": [%q]}`, uuid.New().String()))
					})
					AssertResultFalse()
				})
			})
			Context("flag is enabled with percentage rollout", func() {
				BeforeEach(func() {
					flag.Enabled = true
					err = nil
					flag.Strategies = []data.Strategy{
						{
							Type:    "percentage_rollout",
							Payload: []byte(`{"percentage": 100}`),
						},
					}
				})
				When("100%", func() {
					AssertResultTrue()
				})
				When("0%", func() {
					BeforeEach(func() {
						flag.Strategies[0].Payload = []byte(`{"percentage": 0}`)
					})
					AssertResultFalse()
				})
			})
		})
	})
})
