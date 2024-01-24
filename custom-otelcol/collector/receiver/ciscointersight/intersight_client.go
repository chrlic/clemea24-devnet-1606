package ciscointersight

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/chrlic/otelcol-cust/collector/receiver/ciscointersight/intersightsdk"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type IntersightClient struct {
	ApiClient *intersightsdk.APIClient
	AuthCtx   *context.Context
	logger    *zap.Logger
}

type dialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

func getIntersightSDKClient(config component.Config, logger *zap.Logger) (*IntersightClient, error) {
	// os.Setenv("HTTP_PROXY", "http://proxy_name:proxy_port")

	// ctx := context.WithValue(context.Background(), intersight.ContextServerIndex, 1)

	cfg := config.(*Config)

	apiConfig := intersightsdk.NewConfiguration()
	apiConfig.Host = "intersight.com"
	apiConfig.Scheme = "https"
	apiConfig.Debug = false
	apiClient := intersightsdk.NewAPIClient(apiConfig)

	authConfig := intersightsdk.HttpSignatureAuth{
		KeyId:          cfg.Intersight.ApiKeyId,
		PrivateKeyPath: cfg.Intersight.ApiKeyFile,

		SigningScheme: intersightsdk.HttpSigningSchemeRsaSha256,
		SignedHeaders: []string{
			intersightsdk.HttpSignatureParameterRequestTarget, // The special (request-target) parameter expresses the HTTP request target.
			"Host",   // The Host request header specifies the domain name of the server, and optionally the TCP port number.
			"Date",   // The date and time at which the message was originated.
			"Digest", // A cryptographic digest of the request body.
		},
		SigningAlgorithm: intersightsdk.HttpSigningAlgorithmRsaPKCS1v15,
	}

	authCtx, err := authConfig.ContextWithValue(context.Background())
	if err != nil {
		logger.Sugar().Error("Error creating authentication context - %v", err)
	}
	client := IntersightClient{
		ApiClient: apiClient,
		AuthCtx:   &authCtx,
		logger:    logger,
	}
	return &client, nil
}

// Implement interface jsonscraper.ScrapperClient

func (s *IntersightClient) Login() error {
	return nil
}

func (s *IntersightClient) Logout() error {
	return nil
}

func (s *IntersightClient) DoRequest(method string, uri string, payload *string) (string, error) {
	switch method {
	case "GET":
		return s.intersightGet(uri)
	case "POST":
		return s.intersightPost(uri, payload)
	default:
		return "unimplemented", fmt.Errorf("Method %s not supported", method)
	}
}

func (s *IntersightClient) intersightGet(uri string) (string, error) {

	s.logger.Debug("Intersight GET request", zap.Any("URI", uri))

	response, err := s.ApiClient.DoGet(
		*s.AuthCtx,
		uri,
		"GET",
		map[string]string{},
		url.Values{},
	)

	if err != nil {
		s.logger.Error("Error sending GET to Intersight", zap.Error(err))
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		s.logger.Error("Error reading response from Intersight", zap.Error(err))
		return "", err
	}

	jsonResponse := map[string]interface{}{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		s.logger.Error("Error parsing GET response from Intersight", zap.Error(err))
		return "", err
	}

	// s.logger.Debug("Intersight GET response", zap.Any("GET response body", jsonResponse))

	return string(body), nil
}

func (s *IntersightClient) intersightPost(uri string, payload *string) (string, error) {

	s.logger.Debug("Intersight POST request", zap.Any("URI", uri), zap.Any("payload", *payload))

	response, err := s.ApiClient.DoPost(
		*s.AuthCtx,
		uri,
		"POST",
		payload,
		map[string]string{}, // Intersight SDK detects *string as content type application/json
		url.Values{},
	)

	if err != nil {
		s.logger.Error("Error sending POST to Intersight", zap.Error(err))
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		s.logger.Error("Error reading response from Intersight", zap.Error(err))
		return "", err
	}

	jsonResponse := map[string]interface{}{}
	jsonResponseArr := []interface{}{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		// give it a try with JSON array...
		err = json.Unmarshal(body, &jsonResponseArr)
		if err != nil {
			s.logger.Error("Error parsing POST response from Intersight", zap.Error(err))
			return "", err
		}
	}

	s.logger.Debug("Intersight POST response", zap.Any("POST response body struct", jsonResponse), zap.Any("GET response body arr", jsonResponseArr))

	return string(body), nil
}
