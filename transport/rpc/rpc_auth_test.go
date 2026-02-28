package rpc_test

import (
	"context"
	"net"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nojyerac/go-lib/auth"
	authgrpc "github.com/nojyerac/go-lib/transport/grpc"
	"github.com/nojyerac/semaphore/data"
	mockdata "github.com/nojyerac/semaphore/mocks/data"
	"github.com/nojyerac/semaphore/pb/flag"
	"github.com/nojyerac/semaphore/security"
	. "github.com/nojyerac/semaphore/transport/rpc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const (
	grpcTestIssuer = "semaphore-test"
	grpcTestAud    = "semaphore-api"
	grpcTestSecret = "test-secret"
)

func grpcAuthHeaderForRoles(roles ...string) string {
	claims := jwt.MapClaims{
		"sub":   "grpc-test-user",
		"iss":   grpcTestIssuer,
		"aud":   grpcTestAud,
		"roles": roles,
		"exp":   time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(grpcTestSecret))
	Expect(err).NotTo(HaveOccurred())

	return "Bearer " + signed
}

func grpcCreateFlagRequest() *flag.CreateFlagRequest {
	return &flag.CreateFlagRequest{
		Flag: &flag.Flag{
			Name:    "new-flag",
			Enabled: true,
			Strategies: []*flag.Strategy{
				{
					Type: percentageRolloutStrategyType,
					Payload: &flag.Strategy_PercentageRollout{
						PercentageRollout: &flag.PercentageRollout{Percentage: 50},
					},
				},
			},
		},
	}
}

func withAuthHeader(ctx context.Context, authorization string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", authorization))
}

var _ = Describe("RPC Auth Integration", func() {
	var (
		mockEngine *mockdata.MockDataEngine
		server     *grpc.Server
		listener   *bufconn.Listener
		conn       *grpc.ClientConn
		client     flag.FlagServiceClient
	)

	BeforeEach(func() {
		mockEngine = &mockdata.MockDataEngine{}
		validator := auth.NewValidator(&auth.Configuration{
			Issuer:     grpcTestIssuer,
			Audience:   grpcTestAud,
			HMACSecret: grpcTestSecret,
		})

		server = grpc.NewServer(authgrpc.AuthServerOptions(validator, security.GRPCPolicyMap())...)
		flag.RegisterFlagServiceServer(server, NewFlagService(mockEngine))

		listener = bufconn.Listen(1024 * 1024)
		go func() {
			_ = server.Serve(listener)
		}()

		var err error
		conn, err = grpc.NewClient(
			"passthrough:///bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		Expect(err).NotTo(HaveOccurred())

		client = flag.NewFlagServiceClient(conn)
	})

	AfterEach(func() {
		if conn != nil {
			_ = conn.Close()
		}
		if server != nil {
			server.Stop()
		}
		if listener != nil {
			_ = listener.Close()
		}
		mockEngine.AssertExpectations(GinkgoT())
	})

	It("rejects missing token", func() {
		_, err := client.GetFlag(context.Background(), &flag.GetFlagRequest{Id: uuid.New().String()})
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Unauthenticated))
	})

	It("rejects invalid token", func() {
		ctx := withAuthHeader(context.Background(), "Bearer not-a-jwt")
		_, err := client.GetFlag(ctx, &flag.GetFlagRequest{Id: uuid.New().String()})
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Unauthenticated))
	})

	It("rejects insufficient role", func() {
		ctx := withAuthHeader(context.Background(), grpcAuthHeaderForRoles(security.RoleReader))
		_, err := client.CreateFlag(ctx, grpcCreateFlagRequest())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.PermissionDenied))
	})

	It("allows valid role", func() {
		newID := uuid.New().String()
		mockEngine.On("CreateFlag", mock.Anything, mock.AnythingOfType("*data.FeatureFlag")).Return(newID, nil).Once()

		ctx := withAuthHeader(context.Background(), grpcAuthHeaderForRoles(security.RoleAdmin))
		resp, err := client.CreateFlag(ctx, grpcCreateFlagRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.GetId()).To(Equal(newID))
	})

	It("allows reader role on read operation", func() {
		flagID := uuid.New().String()
		mockEngine.On("GetFlagByID", mock.Anything, flagID).Return((*data.FeatureFlag)(nil), nil).Once()

		ctx := withAuthHeader(context.Background(), grpcAuthHeaderForRoles(security.RoleReader))
		_, err := client.GetFlag(ctx, &flag.GetFlagRequest{Id: flagID})
		Expect(err).NotTo(HaveOccurred())
	})
})
