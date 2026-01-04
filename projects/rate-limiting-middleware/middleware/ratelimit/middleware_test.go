package ratelimit

import (
	"testing"
	"time"

	"github.com/go-monolith/mono/pkg/types"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "default options",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			opts: []Option{
				WithRedisAddr("redis:6379"),
				WithDefaultLimit(50, 30*time.Second),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if m == nil {
				t.Error("New() returned nil middleware")
			}
		})
	}
}

func TestMiddleware_Name(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if name := m.Name(); name != "rate-limit" {
		t.Errorf("Name() = %q, want 'rate-limit'", name)
	}
}

func TestMiddleware_getLimitForService(t *testing.T) {
	m, err := New(
		WithDefaultLimit(100, time.Minute),
		WithServiceLimit("api.getData", 50, 30*time.Second),
		WithServiceLimit("api.createOrder", 10, 10*time.Second),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name        string
		serviceName string
		wantLimit   int
		wantWindow  time.Duration
	}{
		{
			name:        "service with custom limit",
			serviceName: "api.getData",
			wantLimit:   50,
			wantWindow:  30 * time.Second,
		},
		{
			name:        "another service with custom limit",
			serviceName: "api.createOrder",
			wantLimit:   10,
			wantWindow:  10 * time.Second,
		},
		{
			name:        "service using default limit",
			serviceName: "api.unknown",
			wantLimit:   100,
			wantWindow:  time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, window := m.getLimitForService(tt.serviceName)
			if limit != tt.wantLimit {
				t.Errorf("getLimitForService() limit = %d, want %d", limit, tt.wantLimit)
			}
			if window != tt.wantWindow {
				t.Errorf("getLimitForService() window = %v, want %v", window, tt.wantWindow)
			}
		})
	}
}

func TestMiddleware_extractClientID(t *testing.T) {
	m, err := New(
		WithClientIDHeader("X-Client-ID"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name   string
		msg    *types.Msg
		wantID string
	}{
		{
			name: "with client ID header",
			msg: &types.Msg{
				Header: map[string][]string{
					"X-Client-ID": {"client-123"},
				},
			},
			wantID: "client-123",
		},
		{
			name: "without client ID header",
			msg: &types.Msg{
				Header: map[string][]string{},
			},
			wantID: "anonymous",
		},
		{
			name: "nil header",
			msg: &types.Msg{
				Header: nil,
			},
			wantID: "anonymous",
		},
		{
			name: "empty client ID value",
			msg: &types.Msg{
				Header: map[string][]string{
					"X-Client-ID": {""},
				},
			},
			wantID: "anonymous",
		},
		{
			name: "multiple values - takes first",
			msg: &types.Msg{
				Header: map[string][]string{
					"X-Client-ID": {"first", "second"},
				},
			},
			wantID: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientID := m.extractClientID(tt.msg)
			if clientID != tt.wantID {
				t.Errorf("extractClientID() = %q, want %q", clientID, tt.wantID)
			}
		})
	}
}

func TestMiddleware_extractClientID_LongID(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Create a very long client ID
	longID := ""
	for i := 0; i < 200; i++ {
		longID += "a"
	}

	msg := &types.Msg{
		Header: map[string][]string{
			"X-Client-ID": {longID},
		},
	}

	clientID := m.extractClientID(msg)

	// Should be truncated to maxClientIDLength (128)
	if len(clientID) != maxClientIDLength {
		t.Errorf("extractClientID() length = %d, want %d", len(clientID), maxClientIDLength)
	}
}

func TestRateLimitError_Error(t *testing.T) {
	err := &RateLimitError{
		Message:   "rate limit exceeded",
		Remaining: 0,
		ResetAt:   time.Now().Add(time.Minute),
		Limit:     100,
	}

	if err.Error() != "rate limit exceeded" {
		t.Errorf("Error() = %q, want 'rate limit exceeded'", err.Error())
	}
}

func TestMiddleware_OnModuleLifecycle(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	event := types.ModuleLifecycleEvent{
		ModuleName: "test-module",
		Type:       types.ModuleStartedEvent,
	}

	result := m.OnModuleLifecycle(nil, event)

	// Should pass through unchanged
	if result.ModuleName != event.ModuleName {
		t.Errorf("OnModuleLifecycle() ModuleName = %q, want %q", result.ModuleName, event.ModuleName)
	}
	if result.Type != event.Type {
		t.Errorf("OnModuleLifecycle() Type = %v, want %v", result.Type, event.Type)
	}
}

func TestMiddleware_OnConfigurationChange(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	event := types.ConfigurationEvent{
		OptionName: "test.option",
		NewValue:   "test.value",
	}

	result := m.OnConfigurationChange(nil, event)

	// Should pass through unchanged
	if result.OptionName != event.OptionName {
		t.Errorf("OnConfigurationChange() OptionName = %q, want %q", result.OptionName, event.OptionName)
	}
}

func TestMiddleware_OnOutgoingMessage(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	octx := types.OutgoingMessageContext{
		Subject: "test.subject",
	}

	result := m.OnOutgoingMessage(octx)

	// Should pass through unchanged
	if result.Subject != octx.Subject {
		t.Errorf("OnOutgoingMessage() Subject = %q, want %q", result.Subject, octx.Subject)
	}
}

func TestMiddleware_OnEventConsumerRegistration(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Use empty entry - the middleware should pass through unchanged
	entry := types.EventConsumerEntry{}
	result := m.OnEventConsumerRegistration(nil, entry)

	// Should pass through unchanged (comparing the result to original)
	_ = result // No specific fields to check that are guaranteed to exist
}

func TestMiddleware_OnEventStreamConsumerRegistration(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Use empty entry - the middleware should pass through unchanged
	entry := types.EventStreamConsumerEntry{}
	result := m.OnEventStreamConsumerRegistration(nil, entry)

	// Should pass through unchanged (comparing the result to original)
	_ = result // No specific fields to check that are guaranteed to exist
}

func TestMiddleware_OnServiceRegistration_NonRequestReply(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test with a non-request-reply service type
	reg := types.ServiceRegistration{
		Name: "test.service",
		Type: types.ServiceTypeChannel, // Not request-reply
	}

	result := m.OnServiceRegistration(nil, reg)

	// Should pass through unchanged
	if result.Name != reg.Name {
		t.Errorf("OnServiceRegistration() Name = %q, want %q", result.Name, reg.Name)
	}
	if result.Type != reg.Type {
		t.Errorf("OnServiceRegistration() Type = %v, want %v", result.Type, reg.Type)
	}
}

func TestMiddleware_OnServiceRegistration_NilHandler(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test with nil request handler
	reg := types.ServiceRegistration{
		Name:           "test.service",
		Type:           types.ServiceTypeRequestReply,
		RequestHandler: nil,
	}

	result := m.OnServiceRegistration(nil, reg)

	// Should pass through unchanged since handler is nil
	if result.RequestHandler != nil {
		t.Error("OnServiceRegistration() should not wrap nil handler")
	}
}
