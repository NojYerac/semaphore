package http_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nojyerac/go-lib/auth"
	"github.com/nojyerac/go-lib/log"
	libhttp "github.com/nojyerac/go-lib/transport/http"
	"github.com/nojyerac/semaphore/data"
	mockdata "github.com/nojyerac/semaphore/mocks/data"
	"github.com/nojyerac/semaphore/security"
	. "github.com/nojyerac/semaphore/transport/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

const baseURL = "/api/flags"

const (
	testIssuer = "semaphore-test"
	testAud    = "semaphore-api"
	testSecret = "test-secret"
)

func authHeaderForRoles(roles ...string) string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"iss":   testIssuer,
		"aud":   testAud,
		"roles": roles,
		"exp":   time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	Expect(err).NotTo(HaveOccurred())
	return "Bearer " + signed
}

var _ = Describe("Http", func() {
	var (
		mockData *mockdata.MockDataEngine
		srv      libhttp.Server
		req      *http.Request
		method   string
		url      string
		body     io.Reader
		err      error
		resp     *httptest.ResponseRecorder
		flagID   string
		authzHdr string
	)
	BeforeEach(func() {
		flagID = uuid.New().String()
		mockData = &mockdata.MockDataEngine{}
		validator := auth.NewValidator(&auth.Configuration{
			Issuer:     testIssuer,
			Audience:   testAud,
			HMACSecret: testSecret,
		})
		l := log.NewLogger(log.NewConfiguration(), log.WithOutput(GinkgoWriter))
		srv = libhttp.NewServer(
			&libhttp.Configuration{},
			libhttp.WithLogger(l),
			libhttp.WithAuthMiddleware(validator, security.HTTPPolicyMap()),
		)
		authzHdr = authHeaderForRoles(security.RoleAdmin)
		RegisterRoutes(mockData, srv)
	})
	JustBeforeEach(func() {
		req, err = http.NewRequest(method, url, body)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		if authzHdr != "" {
			req.Header.Set("Authorization", authzHdr)
		}
		resp = httptest.NewRecorder()
		srv.ServeHTTP(resp, req)
	})
	AfterEach(func() {
		mockData.AssertExpectations(GinkgoT())
	})
	Describe("GET /livez", func() {
		BeforeEach(func() {
			method = http.MethodGet
			url = "/livez"
			body = http.NoBody
			authzHdr = ""
		})
		It("returns a healthy status", func() {
			Expect(resp.Code).To(Equal(200))
			Expect(resp.Body.String()).To(Equal("ok"))
		})
	})
	Describe("auth middleware", func() {
		BeforeEach(func() {
			method = http.MethodGet
			url = baseURL
			body = http.NoBody
		})

		Context("when token is missing", func() {
			BeforeEach(func() {
				authzHdr = ""
			})

			It("rejects with 401", func() {
				Expect(resp.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when token is invalid", func() {
			BeforeEach(func() {
				authzHdr = "Bearer not-a-jwt"
			})

			It("rejects with 401", func() {
				Expect(resp.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when role is insufficient", func() {
			BeforeEach(func() {
				method = http.MethodPost
				url = baseURL
				body = strings.NewReader(
					`{"name":"flag1","enabled":true,"strategies":[{"type":"percentage_rollout","payload":{"percentage":50}}]}`,
				)
				authzHdr = authHeaderForRoles(security.RoleReader)
			})

			It("rejects with 403", func() {
				Expect(resp.Code).To(Equal(http.StatusForbidden))
			})
		})

		Context("when role is allowed", func() {
			BeforeEach(func() {
				mockData.On("GetFlags", mock.Anything).Return([]*data.FeatureFlag{}, nil).Once()
				authzHdr = authHeaderForRoles(security.RoleReader)
			})

			It("allows request", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})
	})
	Describe("GET /api/flags", func() {
		BeforeEach(func() {
			method = http.MethodGet
			url = baseURL
			body = http.NoBody
			mockData.On("GetFlags", mock.Anything).Return([]*data.FeatureFlag{
				{
					ID:      "flag1",
					Name:    "Flag 1",
					Enabled: true,
					Strategies: []data.Strategy{
						{
							Type:    "percentage_rollout",
							Payload: []byte(`{"percentage": 50}`),
						},
					},
				},
			}, nil).Once()
		})
		It("returns a list of flags", func() {
			Expect(resp.Code).To(Equal(200))
			Expect(resp.Body.String()).To(And(
				ContainSubstring(`"id":"flag1"`),
				ContainSubstring(`"name":"Flag 1"`),
				ContainSubstring(`"enabled":true`),
				ContainSubstring(`"type":"percentage_rollout"`),
				ContainSubstring(`"payload":{"percentage":50}`),
				ContainSubstring(`"createdAt":`),
				ContainSubstring(`"updatedAt":`),
			))
		})
	})
	Describe("POST /api/flags", func() {
		BeforeEach(func() {
			method = http.MethodPost
			url = baseURL
			body = strings.NewReader(
				`{"name":"flag1","enabled":true,"strategies":[{"type":"percentage_rollout","payload":{"percentage":50}}]}`,
			)
			mockData.On("CreateFlag", mock.Anything, mock.AnythingOfType("*data.FeatureFlag")).Return(flagID, nil).Once()
		})
		It("creates a new flag", func() {
			Expect(resp.Code).To(Equal(201))
			Expect(resp.Body.String()).To(ContainSubstring(`"id":"`))
		})
	})
	Describe("GET /api/flags/{id}", func() {
		BeforeEach(func() {
			method = http.MethodGet
			url = baseURL + "/" + flagID
			body = http.NoBody
			mockData.On("GetFlagByID", mock.Anything, flagID).Return(&data.FeatureFlag{
				ID:      flagID,
				Name:    "Flag 1",
				Enabled: true,
				Strategies: []data.Strategy{
					{
						Type:    "percentage_rollout",
						Payload: []byte(`{"percentage": 50}`),
					},
				},
			}, nil).Once()
		})
		It("returns a single flag", func() {
			Expect(resp.Code).To(Equal(200))
			Expect(resp.Body.String()).To(And(
				ContainSubstring(`"id":"`+flagID+`"`),
				ContainSubstring(`"name":"Flag 1"`),
				ContainSubstring(`"enabled":true`),
				ContainSubstring(`"type":"percentage_rollout"`),
				ContainSubstring(`"payload":{"percentage":50}`),
				ContainSubstring(`"createdAt":`),
				ContainSubstring(`"updatedAt":`),
			))
		})
	})
	Describe("PUT /api/flags/{id}", func() {
		BeforeEach(func() {
			method = http.MethodPut
			url = baseURL + "/" + flagID
			body = strings.NewReader(
				`{"name":"flag1","enabled":true,"strategies":[{"type":"percentage_rollout","payload":{"percentage":50}}]}`,
			)
			mockData.On("UpdateFlag", mock.Anything, mock.AnythingOfType("*data.FeatureFlag")).Return(nil).Once()
		})
		It("updates an existing flag", func() {
			Expect(resp.Code).To(Equal(200))
			Expect(resp.Body.String()).To(ContainSubstring(`"id":"` + flagID + `"`))
		})
	})
	Describe("DELETE /api/flags/{id}", func() {
		BeforeEach(func() {
			method = http.MethodDelete
			url = baseURL + "/" + flagID
			body = http.NoBody
			mockData.On("DeleteFlag", mock.Anything, flagID).Return(nil).Once()
		})
		It("deletes a flag", func() {
			Expect(resp.Body.String()).To(MatchJSON(`{"success":true}`))
			Expect(resp.Code).To(Equal(200))
		})
	})
	Describe("POST /api/flags/{id}/evaluate", func() {
		BeforeEach(func() {
			method = http.MethodPost
			url = baseURL + "/" + flagID + "/evaluate"
			body = strings.NewReader(`{"userID":"user1","groupIDs":["group1"]}`)
			mockData.On("EvaluateFlag", mock.Anything, flagID, "user1", []string{"group1"}).Return(true, nil).Once()
		})
		It("evaluates a flag", func() {
			Expect(resp.Code).To(Equal(200))
			Expect(resp.Body.String()).To(ContainSubstring(`"result":true`))
		})
	})
})
