package core

import (
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/yggdrasil-network/yggdrasil-go/src/config"
)

// TestPeerChangeCallback tests that the callback is triggered when peers connect/disconnect
func TestPeerChangeCallback(t *testing.T) {
	// Create two test nodes
	nodeA, err := createTestNode()
	if err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}
	defer nodeA.Stop()

	nodeB, err := createTestNode()
	if err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}
	defer nodeB.Stop()

	// Setup callback tracking for node A
	var mu sync.Mutex
	var callbackCount int
	var lastConnected int
	var lastTotal int

	callback := func(connected, total int) {
		mu.Lock()
		defer mu.Unlock()
		callbackCount++
		lastConnected = connected
		lastTotal = total
		t.Logf("Callback fired: connected=%d, total=%d (call #%d)", connected, total, callbackCount)
	}

	nodeA.SetPeerChangeCallback(callback)

	// Add a peer (should trigger callback with 0 connected, 1 total)
	listenURL := nodeB.links._listeners
	var listenerAddr string
	for listener := range listenURL {
		listenerAddr = listener.Addr().String()
		break
	}

	peerURL, err := url.Parse("tls://" + listenerAddr)
	if err != nil {
		t.Fatalf("Failed to parse peer URL: %v", err)
	}

	// Add peer to node A
	err = nodeA.links.add(peerURL, "", linkTypePersistent)
	if err != nil {
		t.Fatalf("Failed to add peer: %v", err)
	}

	// Wait for callback to be triggered (peer added)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if callbackCount < 1 {
		t.Errorf("Expected at least 1 callback after adding peer, got %d", callbackCount)
	}
	if lastTotal != 1 {
		t.Errorf("Expected total=1 after adding peer, got %d", lastTotal)
	}
	mu.Unlock()

	// Wait for connection to establish (should trigger another callback)
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	connected := false
	for !connected {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for peer connection")
		case <-ticker.C:
			mu.Lock()
			if lastConnected > 0 {
				connected = true
			}
			mu.Unlock()
		}
	}

	mu.Lock()
	if lastConnected != 1 {
		t.Errorf("Expected connected=1 after peer connection, got %d", lastConnected)
	}
	initialCallbackCount := callbackCount
	mu.Unlock()

	t.Logf("Connection established, callback count: %d", initialCallbackCount)

	// Remove the peer (should trigger callback)
	err = nodeA.links.remove(peerURL, "", linkTypePersistent)
	if err != nil {
		t.Fatalf("Failed to remove peer: %v", err)
	}

	// Wait for disconnection callback
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if callbackCount <= initialCallbackCount {
		t.Errorf("Expected callback after removing peer, count stayed at %d", callbackCount)
	}
	mu.Unlock()

	t.Logf("Test completed with %d total callbacks", callbackCount)
}

// TestPeerChangeCallbackNil tests that nil callback doesn't crash
func TestPeerChangeCallbackNil(t *testing.T) {
	node, err := createTestNode()
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	// Set nil callback (should not crash)
	node.SetPeerChangeCallback(nil)

	// Trigger a state change - should not crash even with nil callback
	peerURL, _ := url.Parse("tls://127.0.0.1:9999")
	_ = node.links.add(peerURL, "", linkTypePersistent)

	time.Sleep(100 * time.Millisecond)

	// If we get here without crashing, test passes
	t.Log("Nil callback test passed")
}

// TestPeerChangeCallbackMultipleChanges tests rapid state changes
func TestPeerChangeCallbackMultipleChanges(t *testing.T) {
	node, err := createTestNode()
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	var mu sync.Mutex
	var callbackCount int

	callback := func(connected, total int) {
		mu.Lock()
		callbackCount++
		mu.Unlock()
		t.Logf("Callback: connected=%d, total=%d", connected, total)
	}

	node.SetPeerChangeCallback(callback)

	// Add multiple peers rapidly
	for i := 0; i < 3; i++ {
		peerURL, err := url.Parse(fmt.Sprintf("tls://127.0.0.1:%d", 9000+i))
		if err != nil {
			t.Fatalf("Failed to parse URL: %v", err)
		}
		_ = node.links.add(peerURL, "", linkTypePersistent)
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := callbackCount
	mu.Unlock()

	if count < 3 {
		t.Errorf("Expected at least 3 callbacks, got %d", count)
	}

	t.Logf("Multiple changes test completed with %d callbacks", count)
}

// Helper function to create a test node
func createTestNode() (*Core, error) {
	cfg := config.GenerateConfig()
	if err := cfg.GenerateSelfSignedCertificate(); err != nil {
		return nil, err
	}

	logger := GetLoggerWithPrefix("test", false)

	core, err := New(
		cfg.Certificate,
		logger,
		ListenAddress("tls://127.0.0.1:0"),
	)
	if err != nil {
		return nil, err
	}

	return core, nil
}
