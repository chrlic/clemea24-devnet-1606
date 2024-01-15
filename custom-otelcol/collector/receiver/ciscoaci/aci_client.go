package ciscoaci

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"runtime"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
)

// AciClient - implement interface jsonscraper.ScraperClient

type AciClient struct {
	httpClient *http.Client
	AuthCtx    *context.Context
	logger     *zap.Logger
	config     *AciConfig
	token      string
}

func (s *AciClient) Login() error {
	err := s.login()
	return err
}

func (s *AciClient) Logout() error {
	return s.logout()
}

func (s *AciClient) DoRequest(method string, uri string, payload *string) (string, error) {

	switch method {
	case "GET":
		return s.aciGet(uri)
	default:
		return "", fmt.Errorf("Unimplemented method %s used in ACI Client", method)
	}
}

type dialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

func getAciClient(config component.Config, logger *zap.Logger) (*AciClient, error) {
	cfg := config.(*Config)
	socks5 := cfg.Aci.Socks5

	aciClient := &AciClient{
		config: &cfg.Aci,
		logger: logger,
	}

	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if socks5 != "" {
		dialSocksProxy, err := proxy.SOCKS5("tcp", socks5, nil, baseDialer)
		if err != nil {
			return nil, err
		}

		contextDialer, ok := dialSocksProxy.(proxy.ContextDialer)
		if !ok {
			return nil, err
		}

		httpClient := newClient(contextDialer.DialContext)
		aciClient.httpClient = httpClient
		return aciClient, nil
	} else {
		httpClient := newClient(baseDialer.DialContext)
		aciClient.httpClient = httpClient
		return aciClient, nil
	}
}

func newClient(dialContext dialContextFunc) *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		// error handling
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialContext,
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
		Jar: jar,
	}
}

func (s *AciClient) getHost() string {
	cfg := s.config
	return fmt.Sprintf("%s://%s:%d", cfg.Protocol, cfg.Host, cfg.Port)
}

func (s *AciClient) login() error {
	cfg := s.config
	credsJson := fmt.Sprintf(`
	{
		"aaaUser" : {
		  "attributes" : {
			"name" : "%s",
			"pwd" : "%s"
		  }
		}
	  }	  
	`, cfg.User, cfg.Password)
	creds := bytes.NewBufferString(credsJson)

	response, err := s.httpClient.Post(s.getHost()+"/api/aaaLogin.json", "application/json", creds)
	if err != nil {
		s.logger.Error("Error logging to APIC", zap.Error(err))
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		s.logger.Error("Error reading logging to APIC", zap.Error(err))
		return err
	}

	jsonResponse := map[string]interface{}{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		s.logger.Error("Error parsing login response from APIC", zap.Error(err))
		return err
	}

	// s.logger.Debug("APIC Login response", zap.Any("response body", jsonResponse))

	return nil

}

func (s *AciClient) logout() error {
	// not ending the session anyway, don't do anything
	return nil
}

func (s *AciClient) aciGet(uri string) (string, error) {

	s.logger.Debug("APIC GET request", zap.Any("URI", s.getHost()+uri))

	response, err := s.httpClient.Get(s.getHost() + uri)
	if err != nil {
		s.logger.Error("Error sending GET to APIC", zap.Error(err))
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		s.logger.Error("Error reading response from APIC", zap.Error(err))
	}

	jsonResponse := map[string]interface{}{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		s.logger.Error("Error parsing GET response from APIC", zap.Error(err))
	}

	// s.logger.Debug("APIC GET response", zap.Any("GET response body", jsonResponse))

	return string(body), nil
}
