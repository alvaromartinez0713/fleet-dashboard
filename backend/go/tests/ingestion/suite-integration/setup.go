// +build integration

package suite_integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// IntegrationTestEnv holds configuration for integration tests
type IntegrationTestEnv struct {
	KafkaURL       string
	ZookeeperURL   string
	ToxiproxyURL   string
	ToxiproxyHost  string
	ToxiproxyPort  string
	Cleanup        func()
	ZKContainer    testcontainers.Container
	KafkaContainer testcontainers.Container
	ProxyContainer testcontainers.Container
}

// SetupIntegrationTest initializes Kafka + Zookeeper containers
//
// Usage in tests:
//   env := SetupIntegrationTest(ctx, t)
//   defer env.Cleanup()
//
// Returns KafkaURL like "127.0.0.1:9092"
func SetupIntegrationTest(ctx context.Context, t *testing.T) *IntegrationTestEnv {
	// Skip if Docker is not available
	SkipIfNoDocker(t)

	env := &IntegrationTestEnv{}

	// 1. Start Zookeeper container
	zkReq := testcontainers.ContainerRequest{
		Image:        "confluentinc/cp-zookeeper:7.5.0",
		ExposedPorts: []string{"2181/tcp"},
		Env: map[string]string{
			"ZOOKEEPER_CLIENT_PORT": "2181",
		},
		WaitingFor: wait.ForLog("binding to port").WithStartupTimeout(30 * time.Second),
	}

	zkContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: zkReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start Zookeeper container: %v", err)
	}
	env.ZKContainer = zkContainer

	// Get Zookeeper host:port
	zkHost, err := zkContainer.Host(ctx)
	if err != nil {
		zkContainer.Terminate(ctx)
		t.Fatalf("failed to get Zookeeper host: %v", err)
	}
	zkPort, err := zkContainer.MappedPort(ctx, "2181")
	if err != nil {
		zkContainer.Terminate(ctx)
		t.Fatalf("failed to get Zookeeper port: %v", err)
	}
	env.ZookeeperURL = fmt.Sprintf("%s:%s", zkHost, zkPort.Port())

	// 2. Start Kafka container
	kafkaReq := testcontainers.ContainerRequest{
		Image:        "confluentinc/cp-kafka:7.5.0",
		ExposedPorts: []string{"9092/tcp"},
		Env: map[string]string{
			"KAFKA_BROKER_ID":                     "1",
			"KAFKA_ZOOKEEPER_CONNECT":            env.ZookeeperURL,
			"KAFKA_ADVERTISED_LISTENERS":         fmt.Sprintf("PLAINTEXT://127.0.0.1:9092"),
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP": "PLAINTEXT:PLAINTEXT",
			"KAFKA_INTER_BROKER_LISTENER_NAME":   "PLAINTEXT",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
		},
		WaitingFor: wait.ForLog("started").WithStartupTimeout(60 * time.Second),
	}

	kafkaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: kafkaReq,
		Started:          true,
	})
	if err != nil {
		zkContainer.Terminate(ctx)
		t.Fatalf("failed to start Kafka container: %v", err)
	}
	env.KafkaContainer = kafkaContainer

	// Get Kafka host:port
	kafkaHost, err := kafkaContainer.Host(ctx)
	if err != nil {
		zkContainer.Terminate(ctx)
		kafkaContainer.Terminate(ctx)
		t.Fatalf("failed to get Kafka host: %v", err)
	}
	kafkaPort, err := kafkaContainer.MappedPort(ctx, "9092")
	if err != nil {
		zkContainer.Terminate(ctx)
		kafkaContainer.Terminate(ctx)
		t.Fatalf("failed to get Kafka port: %v", err)
	}
	env.KafkaURL = fmt.Sprintf("%s:%s", kafkaHost, kafkaPort.Port())

	// Wait for Kafka to be fully ready
	if err := WaitForKafkaReady(ctx, env.KafkaURL); err != nil {
		zkContainer.Terminate(ctx)
		kafkaContainer.Terminate(ctx)
		t.Fatalf("timeout waiting for Kafka: %v", err)
	}

	// 3. Register cleanup
	env.Cleanup = func() {
		if env.ZKContainer != nil {
			env.ZKContainer.Terminate(context.Background())
		}
		if env.KafkaContainer != nil {
			env.KafkaContainer.Terminate(context.Background())
		}
		if env.ProxyContainer != nil {
			env.ProxyContainer.Terminate(context.Background())
		}
	}

	return env
}

// SetupKafkaWithFaultsInjection sets up Kafka with Toxiproxy for fault injection
func SetupKafkaWithFaultsInjection(ctx context.Context, t *testing.T) *IntegrationTestEnv {
	// Skip if Docker is not available
	SkipIfNoDocker(t)

	// First setup Kafka normally
	env := SetupIntegrationTest(ctx, t)

	// 3. Start Toxiproxy container
	toxiReq := testcontainers.ContainerRequest{
		Image:        "shopify/toxiproxy:2.5.0",
		ExposedPorts: []string{"8474/tcp", "20000/tcp"},
		WaitingFor:   wait.ForLog("listening on").WithStartupTimeout(30 * time.Second),
	}

	toxiContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: toxiReq,
		Started:          true,
	})
	if err != nil {
		env.Cleanup()
		t.Fatalf("failed to start Toxiproxy container: %v", err)
	}
	env.ProxyContainer = toxiContainer

	// Get Toxiproxy host and ports
	toxiHost, err := toxiContainer.Host(ctx)
	if err != nil {
		env.Cleanup()
		t.Fatalf("failed to get Toxiproxy host: %v", err)
	}
	toxiAPIPort, err := toxiContainer.MappedPort(ctx, "8474")
	if err != nil {
		env.Cleanup()
		t.Fatalf("failed to get Toxiproxy API port: %v", err)
	}
	proxyPort, err := toxiContainer.MappedPort(ctx, "20000")
	if err != nil {
		env.Cleanup()
		t.Fatalf("failed to get Toxiproxy proxy port: %v", err)
	}

	env.ToxiproxyURL = fmt.Sprintf("http://%s:%s", toxiHost, toxiAPIPort.Port())
	env.ToxiproxyHost = toxiHost
	env.ToxiproxyPort = proxyPort.Port()

	// 4. Configure Toxiproxy proxy: listen on 20000 -> forward to Kafka
	// Wait a moment for Toxiproxy to start
	time.Sleep(500 * time.Millisecond)

	// Configure proxy via Toxiproxy REST API
	kafkaHost, _ := env.KafkaContainer.Host(ctx)
	kafkaPort, _ := env.KafkaContainer.MappedPort(ctx, "9092")
	kafkaAddr := fmt.Sprintf("%s:%s", kafkaHost, kafkaPort.Port())

	if err := ConfigureToxiproxyProxy(env.ToxiproxyURL, "kafka-proxy", "0.0.0.0", "20000", kafkaAddr); err != nil {
		env.Cleanup()
		t.Fatalf("failed to configure Toxiproxy: %v", err)
	}

	return env
}

// ConfigureToxiproxyProxy configures a proxy in Toxiproxy via REST API
func ConfigureToxiproxyProxy(toxiproxyURL, name, listen, listenPort, upstreamAddr string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Create proxy JSON payload
	proxyJSON := fmt.Sprintf(`{
		"name": "%s",
		"listen": "[%s]:%s",
		"upstream": "%s"
	}`, name, listen, listenPort, upstreamAddr)

	// POST to Toxiproxy API
	resp, err := client.Post(
		fmt.Sprintf("%s/proxies", toxiproxyURL),
		"application/json",
		io.NopCloser(strings.NewReader(proxyJSON)),
	)
	if err != nil {
		return fmt.Errorf("failed to create Toxiproxy proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Toxiproxy returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SkipIfNoDocker skips test if Docker is not available
func SkipIfNoDocker(t *testing.T) {
	// Try a simple request to verify testcontainers works
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with a simple Alpine container
	req := testcontainers.ContainerRequest{
		Image: "alpine:latest",
	}

	// If GenericContainer fails, Docker is not available
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})

	if err != nil {
		t.Skipf("Docker not available or testcontainers failed: %v", err)
	}

	// Clean up the container we just created
	if container != nil {
		container.Terminate(context.Background())
	}
}

// WaitForKafkaReady waits for Kafka to be ready by attempting connections
func WaitForKafkaReady(ctx context.Context, kafkaURL string) error {
	deadline := time.Now().Add(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Try to connect to Kafka
			conn, err := net.DialTimeout("tcp", kafkaURL, 2*time.Second)
			if err == nil {
				conn.Close()
				return nil // Kafka is ready
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for Kafka at %s", kafkaURL)
			}
		}
	}
}

// CreateKafkaClient creates a Kafka writer for testing
func CreateKafkaClient(brokerURL string) (*kafka.Writer, error) {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{brokerURL},
		Topic:        "positions",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}), nil
}

// ToxiproxyProxy represents a proxy configuration in Toxiproxy
type ToxiproxyProxy struct {
	Name     string `json:"name"`
	Listen   string `json:"listen"`
	Upstream string `json:"upstream"`
}

// ToxicPayload represents a toxic (fault injection) configuration
type ToxicPayload struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Toxicity   float64     `json:"toxicity"`
	Attributes interface{} `json:"attributes,omitempty"`
}

// GetToxicAttributes returns attributes for specific toxic types
func GetToxicAttributes(toxicType string) interface{} {
	switch toxicType {
	case "latency":
		return map[string]int{"latency": 2000} // 2 second latency
	case "timeout":
		return map[string]int{"timeout": 0} // Immediate timeout
	default:
		return nil
	}
}

// AddToxicToProxy adds a toxic (fault injection) to a Toxiproxy proxy via REST API
func AddToxicToProxy(toxiproxyURL, proxyName string, toxic *ToxicPayload) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Marshal the toxic payload to JSON
	toxicJSON, err := json.Marshal(toxic)
	if err != nil {
		return fmt.Errorf("failed to marshal toxic payload: %w", err)
	}

	// POST to Toxiproxy API: POST /proxies/{proxy_name}/toxics
	resp, err := client.Post(
		fmt.Sprintf("%s/proxies/%s/toxics", toxiproxyURL, proxyName),
		"application/json",
		io.NopCloser(strings.NewReader(string(toxicJSON))),
	)
	if err != nil {
		return fmt.Errorf("failed to add toxic to proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Toxiproxy returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
