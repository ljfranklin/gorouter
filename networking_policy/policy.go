package networking_policy

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/gorouter/logger"

	"github.com/uber-go/zap"

	"code.cloudfoundry.org/gorouter/config"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/go-db-helpers/json_client"
	"code.cloudfoundry.org/go-db-helpers/mutualtls"
)

type Source struct {
	ID  string `json:"id"`
	Tag string `json:"tag,omitempty"`
}

type Destination struct {
	ID       string `json:"id"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}
type Policy struct {
	Source      Source      `json:"source"`
	Destination Destination `json:"destination"`
}

var policies struct {
	Policies []Policy `json:"policies"`
}

type PolicyClientConfig struct {
	networkPolicyServerConfig config.NetworkPolicyServerConfig
	// using lager just for policy client
	logger    lager.Logger
	tlsConfig *tls.Config
	// use zlogger for all error and info logging
	zlogger logger.Logger
}

func NewPolicyClientConfig(networkPolicyServer config.NetworkPolicyServerConfig, zlogger logger.Logger) *PolicyClientConfig {
	policyClientLogger := lager.NewLogger("network-policy-client")
	policyClientLogger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	if (networkPolicyServer.ClientCertFile != "") &&
		networkPolicyServer.ClientKeyFile != "" && networkPolicyServer.ServerCACertFile != "" {

		clientCertFile, _ := ioutil.TempFile("", "clientCert")
		_, err := clientCertFile.Write([]byte(networkPolicyServer.ClientCertFile))
		if err != nil {
			zlogger.Fatal("client-cert-file-write-error", zap.Error(err))
		}
		clientCertFile.Close()

		clientKeyFile, _ := ioutil.TempFile("", "clientKey")
		clientKeyFile.Write([]byte(networkPolicyServer.ClientKeyFile))
		clientKeyFile.Close()

		serverCACertFile, _ := ioutil.TempFile("", "serverCACert")
		serverCACertFile.Write([]byte(networkPolicyServer.ServerCACertFile))
		serverCACertFile.Close()

		clientTLSConfig, err := mutualtls.NewClientTLSConfig(
			clientCertFile.Name(),
			clientKeyFile.Name(),
			serverCACertFile.Name())
		if err != nil {
			policyClientLogger.Fatal("mutual tls config", err)
		}
		return &PolicyClientConfig{
			tlsConfig: clientTLSConfig,
			logger:    policyClientLogger,
			networkPolicyServerConfig: networkPolicyServer,
			zlogger:                   zlogger,
		}
	}
	return nil
}

// Runs the ifrit process
func (p *PolicyClientConfig) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	ticker := time.NewTicker(time.Millisecond * 500)

	networkPolicyHTTPClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: p.tlsConfig,
		},
		Timeout: time.Duration(5) * time.Second,
	}
	policyClient := json_client.New(
		p.logger,
		networkPolicyHTTPClient,
		fmt.Sprintf("https://%s:%d", p.networkPolicyServerConfig.Host, p.networkPolicyServerConfig.Port),
	)
	close(ready)
	var err error
	for {
		select {
		case <-ticker.C:
			err = policyClient.Do("GET", "/networking/v0/internal/policies", nil, &policies, "")
			if err != nil {
				p.zlogger.Fatal("policy-client-error", zap.Error(err))
			}
			p.zlogger.Info("got-polices", zap.Object("policies", policies))
		case <-signals:
			return nil
		}
	}
}