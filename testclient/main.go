package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/nojyerac/go-lib/log"
	libgrpc "github.com/nojyerac/go-lib/transport/grpc"
	"github.com/nojyerac/semaphore/pb/flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
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
	stream, err := flagClient.ListFlags(ctx, &flag.ListFlagsRequest{})
	if err != nil {
		panic(err)
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				logger.Info("flag stream closed by server")
				break
			}
			panic(err)
		}
		logger.Infof("Received flag: %s", resp.Flag.Name)
	}

	if statusCode, body, err := do("GET", "http://localhost:8080/api/flags", http.NoBody); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", string(body))
	}
	createFlagBody := `{
		"name": "new-feature",
		"description": "A new feature flag",
		"enabled": true,
		"strategies": [
			{
				"type": "percentage_rollout",
				"payload": {
					"percentage": 50
				}
			}
		]
	}`
	var createdFlagID string
	if statusCode, body, err := do("POST", "http://localhost:8080/api/flags", strings.NewReader(createFlagBody)); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", string(body))
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
	if statusCode, body, err := do("GET", "http://localhost:8080/api/flags/"+createdFlagID, http.NoBody); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", string(body))
	}
	evaluateFlagBody := fmt.Sprintf(`{
		"userId": %q,
		"groupIds": [%q, %q]
	}`, uuid.New().String(), uuid.New().String(), uuid.New().String())
	if statusCode, body, err := do("POST", "http://localhost:8080/api/flags/"+createdFlagID+"/evaluate", strings.NewReader(evaluateFlagBody)); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", string(body))
	}
	updateFlagBody := `{
		"name": "new-feature-updated",
		"description": "An updated feature flag",
		"enabled": false,
		"strategies": []
	}`
	if statusCode, body, err := do("PUT", "http://localhost:8080/api/flags/"+createdFlagID, strings.NewReader(updateFlagBody)); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", string(body))
	}
	if statusCode, body, err := do("DELETE", "http://localhost:8080/api/flags/"+createdFlagID, http.NoBody); err != nil {
		logger.WithError(err).Error("failed to make HTTP request")
	} else {
		logger.WithField("status_code", statusCode).Infof("Received HTTP response: %s", string(body))
	}
}

func do(method, url string, body io.Reader) (int, string, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, "", err
	}
	return res.StatusCode, string(bodyBytes), nil
}
