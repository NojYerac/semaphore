package rpc_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nojyerac/semaphore/data"
	mockdata "github.com/nojyerac/semaphore/mocks/data"
	"github.com/nojyerac/semaphore/pb/flag"
	. "github.com/nojyerac/semaphore/transport/rpc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("RPC Service", func() {
	var (
		service    *FlagService
		mockEngine *mockdata.MockDataEngine
		ctx        context.Context
	)

	BeforeEach(func() {
		mockEngine = &mockdata.MockDataEngine{}
		service = NewFlagService(mockEngine)
		ctx = context.Background()
	})

	AfterEach(func() {
		mockEngine.AssertExpectations(GinkgoT())
	})

	Describe("GetFlag", func() {
		var (
			req      *flag.GetFlagRequest
			resp     *flag.GetFlagResponse
			err      error
			flagID   string
			flagName string
		)

		BeforeEach(func() {
			flagID = uuid.New().String()
			flagName = "test-flag"
			req = &flag.GetFlagRequest{Id: flagID}
		})

		Context("when flag exists", func() {
			BeforeEach(func() {
				f := &data.FeatureFlag{
					ID:      flagID,
					Name:    flagName,
					Enabled: true,
				}
				mockEngine.On("GetFlagByID", mock.Anything, flagID).Return(f, nil)
			})

			It("returns the flag", func() {
				resp, err = service.GetFlag(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Flag.Id).To(Equal(flagID))
				Expect(resp.Flag.Name).To(Equal(flagName))
			})
		})

		Context("when flag does not exist", func() {
			BeforeEach(func() {
				mockEngine.On("GetFlagByID", mock.Anything, flagID).Return(nil, nil)
			})

			It("returns nil response (or handled error)", func() {
				resp, err = service.GetFlag(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeNil())
			})
		})

		Context("when db error occurs", func() {
			BeforeEach(func() {
				mockEngine.On("GetFlagByID", mock.Anything, flagID).Return(nil, fmt.Errorf("db error"))
			})

			It("returns error", func() {
				resp, err = service.GetFlag(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("db error"))
			})
		})
	})

	Describe("CreateFlag", func() {
		var (
			req   *flag.CreateFlagRequest
			resp  *flag.CreateFlagResponse
			err   error
			newID string
		)

		BeforeEach(func() {
			newID = uuid.New().String()
			req = &flag.CreateFlagRequest{
				Flag: &flag.Flag{
					Name:    "new-flag",
					Enabled: true,
					Strategies: []*flag.Strategy{
						{
							Type: "percentage_rollout",
							Payload: &flag.Strategy_PercentageRollout{
								PercentageRollout: &flag.PercentageRollout{Percentage: 50},
							},
						},
					},
				},
			}
		})

		It("creates the flag", func() {
			// We expect CreateFlag to be called with a converted struct
			mockEngine.On("CreateFlag", mock.Anything, mock.MatchedBy(func(f *data.FeatureFlag) bool {
				return f.Name == "new-flag" &&
					f.Enabled == true &&
					len(f.Strategies) == 1 &&
					f.Strategies[0].Type == "percentage_rollout"
			})).Return(newID, nil)

			resp, err = service.CreateFlag(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Id).To(Equal(newID))
		})
	})

	Describe("Evaluate", func() {
		var (
			req    *flag.EvaluateRequest
			resp   *flag.EvaluateResponse
			err    error
			flagID string
			userID string
		)

		BeforeEach(func() {
			flagID = uuid.New().String()
			userID = uuid.New().String()
			req = &flag.EvaluateRequest{
				FlagId:   flagID,
				UserId:   userID,
				GroupIds: []string{"group1"},
			}
		})

		It("evaluates the flag", func() {
			mockEngine.On("EvaluateFlag", mock.Anything, flagID, userID, []string{"group1"}).Return(true, nil)

			resp, err = service.Evaluate(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Enabled).To(BeTrue())
		})
	})
})
