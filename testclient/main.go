//nolint

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nojyerac/go-lib/log"
	libgrpc "github.com/nojyerac/go-lib/transport/grpc"
	"github.com/nojyerac/semaphore/pb/flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	baseURL    = "http://localhost:8080/api/flags"
	roleReader = "flag_reader"
	roleAdmin  = "flag_admin"
)

func main() { // nolint
	logger := log.NewLogger(&log.Configuration{
		LogLevel:     "debug",
		HumanFrendly: true,
	}).WithField("service", "testclient")
	libgrpc.SetLogger(logger)
	ctx := log.WithLogger(context.Background(), logger)
	log.SetDefaultCtxLogger(logger)

	creds := insecure.NewCredentials()
	cc, err := libgrpc.ClientConn(
		"localhost:8080",
		libgrpc.WithDialOptions(grpc.WithTransportCredentials(creds)),
	)
	if err != nil {
		panic(err)
	}
	defer cc.Close()
	flagClient := flag.NewFlagServiceClient(cc)
	readToken, err := issueToken(roleReader)
	if err != nil {
		panic(err)
	}
	adminToken, err := issueToken(roleAdmin)
	if err != nil {
		panic(err)
	}
	readCtx := withBearer(ctx, readToken)
	adminCtx := withBearer(ctx, adminToken)

	var createdGrpcFlagID string
	if res, err := flagClient.CreateFlag(adminCtx, &flag.CreateFlagRequest{
		Flag: &flag.Flag{
			Name:        "new-grpc-flag",
			Enabled:     true,
			Description: "A new flag (gRPC)",
			Strategies: []*flag.Strategy{{
				Type: "percentage_rollout",
				Payload: &flag.Strategy_PercentageRollout{
					PercentageRollout: &flag.PercentageRollout{
						Percentage: 50,
					},
				},
			}},
		},
	}); err != nil {
		logger.WithError(err).Error("failed to create flag via gRPC")
	} else {
		logger.Infof("Created flag with ID: %s", res.GetId())
		createdGrpcFlagID = res.GetId()
	}
	stream, err := flagClient.ListFlags(readCtx, &flag.ListFlagsRequest{})
	if err != nil {
		logger.WithError(err).Error("failed to list flags via gRPC")
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				logger.Info("flag stream closed by server")
				break
			}
			logger.WithError(err).Error("failed to receive flag from stream")
		}
		logger.Infof("Received flag: %+v", resp.Flag)
	}

	if getRes, err := flagClient.GetFlag(readCtx, &flag.GetFlagRequest{Id: createdGrpcFlagID}); err != nil {
		logger.WithError(err).Error("failed to get flag via gRPC")
	} else {
		logger.Infof("Got flag via gRPC: %s", getRes.GetFlag().GetName())
	}
	if _, err := flagClient.UpdateFlag(adminCtx, &flag.UpdateFlagRequest{
		Flag: &flag.Flag{
			Id:          createdGrpcFlagID,
			Name:        "updated-grpc-flag",
			Enabled:     false,
			Description: "An updated flag (gRPC)",
			Strategies:  []*flag.Strategy{},
		},
	}); err != nil {
		logger.WithError(err).Error("failed to update flag via gRPC")
	} else {
		logger.Info("Updated flag via gRPC")
	}
	if evalRes, err := flagClient.Evaluate(readCtx, &flag.EvaluateRequest{
		FlagId:   createdGrpcFlagID,
		UserId:   uuid.New().String(),
		GroupIds: []string{uuid.New().String(), uuid.New().String()},
	}); err != nil {
		logger.WithError(err).Error("failed to evaluate flag via gRPC")
	} else {
		logger.Infof("Evaluated flag via gRPC, enabled: %v", evalRes.GetEnabled())
	}
	if _, err := flagClient.DeleteFlag(adminCtx, &flag.DeleteFlagRequest{Id: createdGrpcFlagID}); err != nil {
		logger.WithError(err).Error("failed to delete flag via gRPC")
	} else {
		logger.Info("Deleted flag via gRPC")
	}

	// Test HTTP endpoints
	if statusCode, body, err := do("GET", baseURL, http.NoBody, readToken); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", body)
	}
	createFlagBody := `{
		"name": "new-feature",
		"description": "A new feature flag",
		"enabled": true,
		"strategies": [{
			"type": "percentage_rollout",
			"payload": {"percentage": 50}
		}]}`
	var createdFlagID string
	if statusCode, body, err := do("POST", baseURL, strings.NewReader(createFlagBody), adminToken); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", body)
		createdFlagBody := struct {
			ID string `json:"id"`
		}{}
		if err := json.Unmarshal([]byte(body), &createdFlagBody); err != nil {
			logger.WithError(err).Error("failed to unmarshal created flag response")
		} else {
			createdFlagID = createdFlagBody.ID
			logger.WithField("flag_id", createdFlagID).Info("Created flag with ID")
		}
	}
	if createdFlagID == "" {
		logger.Error("created flag ID is empty, skipping GET and DELETE tests")
		return
	}
	if statusCode, body, err := do("GET", baseURL+"/"+createdFlagID, http.NoBody, readToken); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", body)
	}
	evaluateFlagBody := fmt.Sprintf(
		`{"userId": %q,"groupIds": [%q, %q]}`,
		uuid.New().String(), uuid.New().String(), uuid.New().String(),
	)
	evaluateFlagBodyReader := strings.NewReader(evaluateFlagBody)
	if statusCode, body, err := do(
		"POST", baseURL+"/"+createdFlagID+"/evaluate", evaluateFlagBodyReader, readToken,
	); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", body)
	}
	updateFlagBody := `{
		"name": "new-feature-updated",
		"description": "An updated feature flag",
		"enabled": false,
		"strategies": []
	}`
	if statusCode, body, err := do("PUT", baseURL+"/"+createdFlagID, strings.NewReader(updateFlagBody), adminToken); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", body)
	}
	if statusCode, body, err := do("DELETE", baseURL+"/"+createdFlagID, http.NoBody, adminToken); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", body)
	}
}

func do(method, url string, body io.Reader, bearerToken string) (code int, bodyStr string, err error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, "", err
	}
	return res.StatusCode, string(bodyBytes), nil
}

func withBearer(ctx context.Context, token string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
}

func issueToken(roles ...string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   getenv("TESTCLIENT_AUTH_SUBJECT", "testclient"),
		"iss":   getenv("TESTCLIENT_AUTH_ISSUER", "semaphore"),
		"aud":   getenv("TESTCLIENT_AUTH_AUDIENCE", "semaphore-api"),
		"roles": roles,
		"iat":   now.Unix(),
		"exp":   now.Add(1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getenv("TESTCLIENT_AUTH_HMAC_SECRET", "change-me")))
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
