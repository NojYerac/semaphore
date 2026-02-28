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
	"google.golang.org/grpc/metadata"
)

type listFlagsServerStub struct {
	ctx       context.Context
	responses []*flag.ListFlagsResponse
	sendErr   error
}

func (s *listFlagsServerStub) Send(resp *flag.ListFlagsResponse) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.responses = append(s.responses, resp)
	return nil
}

func (s *listFlagsServerStub) SetHeader(metadata.MD) error { return nil }

func (s *listFlagsServerStub) SendHeader(metadata.MD) error { return nil }

func (s *listFlagsServerStub) SetTrailer(metadata.MD) {}

func (s *listFlagsServerStub) Context() context.Context { return s.ctx }

func (s *listFlagsServerStub) SendMsg(interface{}) error { return nil }

func (s *listFlagsServerStub) RecvMsg(interface{}) error { return nil }

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

	Describe("ListFlags", func() {
		var (
			req      *flag.ListFlagsRequest
			srv      *listFlagsServerStub
			err      error
			firstID  string
			secondID string
		)

		BeforeEach(func() {
			req = &flag.ListFlagsRequest{}
			srv = &listFlagsServerStub{ctx: ctx}
			firstID = uuid.New().String()
			secondID = uuid.New().String()
		})

		Context("when flags are returned", func() {
			BeforeEach(func() {
				mockEngine.On("GetFlags", mock.Anything).Return([]*data.FeatureFlag{
					{ID: firstID, Name: "flag-1", Enabled: true},
					{ID: secondID, Name: "flag-2", Enabled: false},
				}, nil)
			})

			It("streams all flags", func() {
				err = service.ListFlags(req, srv)
				Expect(err).NotTo(HaveOccurred())
				Expect(srv.responses).To(HaveLen(2))
				Expect(srv.responses[0].GetFlag().GetId()).To(Equal(firstID))
				Expect(srv.responses[1].GetFlag().GetId()).To(Equal(secondID))
			})
		})

		Context("when source returns an error", func() {
			BeforeEach(func() {
				mockEngine.On("GetFlags", mock.Anything).Return(nil, fmt.Errorf("list failed"))
			})

			It("returns the source error", func() {
				err = service.ListFlags(req, srv)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("list failed"))
			})
		})

		Context("when sending a response fails", func() {
			BeforeEach(func() {
				srv.sendErr = fmt.Errorf("stream send failed")
				mockEngine.On("GetFlags", mock.Anything).Return([]*data.FeatureFlag{
					{ID: firstID, Name: "flag-1", Enabled: true},
				}, nil)
			})

			It("returns the stream error", func() {
				err = service.ListFlags(req, srv)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("stream send failed"))
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

	Describe("UpdateFlag", func() {
		var (
			req    *flag.UpdateFlagRequest
			resp   *flag.UpdateFlagResponse
			err    error
			flagID string
		)

		BeforeEach(func() {
			flagID = uuid.New().String()
			req = &flag.UpdateFlagRequest{
				Flag: &flag.Flag{
					Id:      flagID,
					Name:    "updated-flag",
					Enabled: true,
					Strategies: []*flag.Strategy{
						{
							Type: "percentage_rollout",
							Payload: &flag.Strategy_PercentageRollout{
								PercentageRollout: &flag.PercentageRollout{Percentage: 75},
							},
						},
					},
				},
			}
		})

		It("updates the flag", func() {
			mockEngine.On("UpdateFlag", mock.Anything, mock.MatchedBy(func(f *data.FeatureFlag) bool {
				return f.ID == flagID &&
					f.Name == "updated-flag" &&
					f.Enabled &&
					len(f.Strategies) == 1 &&
					f.Strategies[0].Type == "percentage_rollout"
			})).Return(nil)

			resp, err = service.UpdateFlag(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
			Expect(resp.Success).To(BeTrue())
		})

		Context("when the request flag is invalid", func() {
			BeforeEach(func() {
				req = &flag.UpdateFlagRequest{
					Flag: &flag.Flag{
						Id:   flagID,
						Name: "updated-flag",
						Strategies: []*flag.Strategy{{
							Type: "percentage_rollout",
						}},
					},
				}
			})

			It("returns conversion error", func() {
				resp, err = service.UpdateFlag(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(resp).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("unknown strategy payload type"))
			})
		})

		Context("when source update fails", func() {
			BeforeEach(func() {
				mockEngine.On("UpdateFlag", mock.Anything, mock.AnythingOfType("*data.FeatureFlag")).Return(fmt.Errorf("update failed"))
			})

			It("returns the source error", func() {
				resp, err = service.UpdateFlag(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(resp).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("update failed"))
			})
		})
	})

	Describe("DeleteFlag", func() {
		var (
			req    *flag.DeleteFlagRequest
			resp   *flag.DeleteFlagResponse
			err    error
			flagID string
		)

		BeforeEach(func() {
			flagID = uuid.New().String()
			req = &flag.DeleteFlagRequest{Id: flagID}
		})

		It("deletes the flag", func() {
			mockEngine.On("DeleteFlag", mock.Anything, flagID).Return(nil)

			resp, err = service.DeleteFlag(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
			Expect(resp.Success).To(BeTrue())
		})

		Context("when source delete fails", func() {
			BeforeEach(func() {
				mockEngine.On("DeleteFlag", mock.Anything, flagID).Return(fmt.Errorf("delete failed"))
			})

			It("returns the source error", func() {
				resp, err = service.DeleteFlag(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(resp).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("delete failed"))
			})
		})
	})
})
