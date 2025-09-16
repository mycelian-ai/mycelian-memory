package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmbedder is a mock implementation of EmbeddingProvider
type MockEmbedder struct {
	mock.Mock
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

// MockIndex is a mock implementation of SearchIndex
type MockIndex struct {
	mock.Mock
}

func (m *MockIndex) Search(ctx context.Context, actorID, memoryID, query string, vec []float32, topKE int, alpha float32, includeRawEntries bool) ([]model.SearchHit, error) {
	args := m.Called(ctx, actorID, memoryID, query, vec, topKE, alpha, includeRawEntries)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.SearchHit), args.Error(1)
}

func (m *MockIndex) LatestContext(ctx context.Context, actorID, memoryID string) (string, time.Time, error) {
	args := m.Called(ctx, actorID, memoryID)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *MockIndex) SearchContexts(ctx context.Context, actorID, memoryID, query string, vec []float32, topKC int, alpha float32) ([]model.ContextHit, error) {
	args := m.Called(ctx, actorID, memoryID, query, vec, topKC, alpha)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ContextHit), args.Error(1)
}

func (m *MockIndex) UpsertEntry(ctx context.Context, id string, vector []float32, payload map[string]interface{}) error {
	args := m.Called(ctx, id, vector, payload)
	return args.Error(0)
}

func (m *MockIndex) DeleteEntry(ctx context.Context, actorID, id string) error {
	args := m.Called(ctx, actorID, id)
	return args.Error(0)
}

func (m *MockIndex) UpsertContext(ctx context.Context, id string, vector []float32, payload map[string]interface{}) error {
	args := m.Called(ctx, id, vector, payload)
	return args.Error(0)
}

func (m *MockIndex) DeleteContext(ctx context.Context, actorID, id string) error {
	args := m.Called(ctx, actorID, id)
	return args.Error(0)
}

func (m *MockIndex) DeleteMemory(ctx context.Context, actorID, memoryID string) error {
	args := m.Called(ctx, actorID, memoryID)
	return args.Error(0)
}

func (m *MockIndex) DeleteVault(ctx context.Context, actorID, vaultID string) error {
	args := m.Called(ctx, actorID, vaultID)
	return args.Error(0)
}

// Helper function to create a test worker with mocks
func createTestWorker() (*Worker, *MockEmbedder, *MockIndex) {
	mockEmbed := &MockEmbedder{}
	mockIndex := &MockIndex{}
	logger := zerolog.New(nil).With().Logger() // Silent logger for tests
	cfg := Config{BatchSize: 10, Interval: 0}

	worker := &Worker{
		db:       nil, // We're testing handle() which doesn't use db
		log:      logger,
		embedder: mockEmbed,
		index:    mockIndex,
		cfg:      cfg,
	}

	return worker, mockEmbed, mockIndex
}

// ============================================================================
// Empty Text Handling Tests (Core Fix Validation)
// ============================================================================

func TestHandleSkipsEmptyEntry_BothEmpty(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	job := job{
		id:          1,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary":  "",
			"rawEntry": "",
		},
	}

	err := worker.handle(ctx, job)

	// Should succeed without calling embed or index
	assert.NoError(t, err)
	mockEmbed.AssertNotCalled(t, "Embed")
	mockIndex.AssertNotCalled(t, "UpsertEntry")
}

func TestHandleSkipsEmptyEntry_EmptySummaryWhitespaceRawEntry(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	job := job{
		id:          2,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary":  "",
			"rawEntry": "   \t\n  ", // Various whitespace
		},
	}

	err := worker.handle(ctx, job)

	// Should succeed without calling embed or index
	assert.NoError(t, err)
	mockEmbed.AssertNotCalled(t, "Embed")
	mockIndex.AssertNotCalled(t, "UpsertEntry")
}

func TestHandleSkipsEmptyEntry_WhitespaceSummaryEmptyRawEntry(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	job := job{
		id:          3,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary":  "\n\n\t ",
			"rawEntry": "",
		},
	}

	err := worker.handle(ctx, job)

	// Should succeed without calling embed or index
	assert.NoError(t, err)
	mockEmbed.AssertNotCalled(t, "Embed")
	mockIndex.AssertNotCalled(t, "UpsertEntry")
}

func TestHandleUsesPreferredText_BothPresent(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	expectedVector := make([]float32, 768)
	for i := range expectedVector {
		expectedVector[i] = float32(i) / 768.0
	}

	job := job{
		id:          4,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary":  "This is the summary",
			"rawEntry": "This is the raw entry",
		},
	}

	// Should use summary (preferred) not rawEntry
	mockEmbed.On("Embed", ctx, "This is the summary").Return(expectedVector, nil)
	mockIndex.On("UpsertEntry", ctx, "test-entry-id", expectedVector, job.payload).Return(nil)

	err := worker.handle(ctx, job)

	assert.NoError(t, err)
	mockEmbed.AssertCalled(t, "Embed", ctx, "This is the summary")
	mockIndex.AssertCalled(t, "UpsertEntry", ctx, "test-entry-id", expectedVector, job.payload)
}

// ============================================================================
// Embedding Service Failure Tests
// ============================================================================

func TestHandleEmbeddingError_ReturnsError(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	embedError := errors.New("embedding service unavailable")

	job := job{
		id:          5,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary": "Valid text to embed",
		},
	}

	mockEmbed.On("Embed", ctx, "Valid text to embed").Return(nil, embedError)

	err := worker.handle(ctx, job)

	// Should return the embedding error
	assert.Error(t, err)
	assert.Equal(t, embedError, err)
	mockIndex.AssertNotCalled(t, "UpsertEntry")
}

func TestHandleEmbeddingError_WrongDimension(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	wrongVector := []float32{0.5} // 1-dimensional instead of 768

	job := job{
		id:          6,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary": "Text causing wrong dimension",
		},
	}

	mockEmbed.On("Embed", ctx, "Text causing wrong dimension").Return(wrongVector, nil)
	indexError := errors.New("vector dimension mismatch: expected 768, got 1")
	mockIndex.On("UpsertEntry", ctx, "test-entry-id", wrongVector, job.payload).Return(indexError)

	err := worker.handle(ctx, job)

	// Should return the index error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dimension")
}

// ============================================================================
// Index Service Failure Tests
// ============================================================================

func TestHandleIndexError_GenericError(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	expectedVector := make([]float32, 768)
	indexError := errors.New("connection refused")

	job := job{
		id:          7,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary": "Valid text",
		},
	}

	mockEmbed.On("Embed", ctx, "Valid text").Return(expectedVector, nil)
	mockIndex.On("UpsertEntry", ctx, "test-entry-id", expectedVector, job.payload).Return(indexError)

	err := worker.handle(ctx, job)

	// Should return the index error
	assert.Error(t, err)
	assert.Equal(t, indexError, err)
}

func TestHandleIndexError_AlreadyExists(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	expectedVector := make([]float32, 768)
	alreadyExistsError := errors.New("object already exists in index")

	job := job{
		id:          8,
		op:          OpUpsertEntry,
		aggregateID: "existing-entry-id",
		payload: map[string]interface{}{
			"summary": "Duplicate entry",
		},
	}

	mockEmbed.On("Embed", ctx, "Duplicate entry").Return(expectedVector, nil)
	mockIndex.On("UpsertEntry", ctx, "existing-entry-id", expectedVector, job.payload).Return(alreadyExistsError)

	err := worker.handle(ctx, job)

	// Should return nil (mark as done) for already exists
	assert.NoError(t, err)
}

func TestHandleIndexError_Status422(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	expectedVector := make([]float32, 768)
	status422Error := errors.New("status code: 422 unprocessable entity")

	job := job{
		id:          9,
		op:          OpUpsertEntry,
		aggregateID: "duplicate-entry-id",
		payload: map[string]interface{}{
			"summary": "Another duplicate",
		},
	}

	mockEmbed.On("Embed", ctx, "Another duplicate").Return(expectedVector, nil)
	mockIndex.On("UpsertEntry", ctx, "duplicate-entry-id", expectedVector, job.payload).Return(status422Error)

	err := worker.handle(ctx, job)

	// Should return nil (mark as done) for 422 status
	assert.NoError(t, err)
}

// ============================================================================
// Malformed Payload Tests
// ============================================================================

func TestHandleBadPayload_MissingSummary(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	expectedVector := make([]float32, 768)

	job := job{
		id:          10,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			// summary field missing
			"rawEntry": "Fallback to raw entry",
		},
	}

	// Should use rawEntry when summary is missing
	mockEmbed.On("Embed", ctx, "Fallback to raw entry").Return(expectedVector, nil)
	mockIndex.On("UpsertEntry", ctx, "test-entry-id", expectedVector, job.payload).Return(nil)

	err := worker.handle(ctx, job)

	assert.NoError(t, err)
	mockEmbed.AssertCalled(t, "Embed", ctx, "Fallback to raw entry")
}

func TestHandleBadPayload_MissingRawEntry(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	expectedVector := make([]float32, 768)

	job := job{
		id:          11,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary": "Only summary present",
			// rawEntry field missing
		},
	}

	// Should use summary when rawEntry is missing
	mockEmbed.On("Embed", ctx, "Only summary present").Return(expectedVector, nil)
	mockIndex.On("UpsertEntry", ctx, "test-entry-id", expectedVector, job.payload).Return(nil)

	err := worker.handle(ctx, job)

	assert.NoError(t, err)
	mockEmbed.AssertCalled(t, "Embed", ctx, "Only summary present")
}

func TestHandleBadPayload_BothMissing(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	job := job{
		id:          12,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			// Both summary and rawEntry missing
			"otherField": "some value",
		},
	}

	err := worker.handle(ctx, job)

	// Should succeed without calling embed or index
	assert.NoError(t, err)
	mockEmbed.AssertNotCalled(t, "Embed")
	mockIndex.AssertNotCalled(t, "UpsertEntry")
}

func TestHandleBadPayload_NonStringValues(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	job := job{
		id:          13,
		op:          OpUpsertEntry,
		aggregateID: "test-entry-id",
		payload: map[string]interface{}{
			"summary":  123,                                // Number instead of string
			"rawEntry": []string{"array", "of", "strings"}, // Array instead of string
		},
	}

	err := worker.handle(ctx, job)

	// Should succeed without calling embed or index (treated as empty)
	assert.NoError(t, err)
	mockEmbed.AssertNotCalled(t, "Embed")
	mockIndex.AssertNotCalled(t, "UpsertEntry")
}

// ============================================================================
// Unknown Operation Tests
// ============================================================================

func TestHandleUnknownOp_InvalidOp(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	job := job{
		id:          14,
		op:          "invalid_operation",
		aggregateID: "test-id",
		payload:     map[string]interface{}{},
	}

	err := worker.handle(ctx, job)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown op")
	mockEmbed.AssertNotCalled(t, "Embed")
	mockIndex.AssertNotCalled(t, "UpsertEntry")
}

func TestHandleUnknownOp_EmptyOp(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	job := job{
		id:          15,
		op:          "",
		aggregateID: "test-id",
		payload:     map[string]interface{}{},
	}

	err := worker.handle(ctx, job)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown op")
	mockEmbed.AssertNotCalled(t, "Embed")
	mockIndex.AssertNotCalled(t, "UpsertEntry")
}

// ============================================================================
// Context Operation Tests (similar to Entry tests)
// ============================================================================

func TestHandleSkipsEmptyContext(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	job := job{
		id:          16,
		op:          OpUpsertContext,
		aggregateID: "test-context-id",
		payload: map[string]interface{}{
			"context": "  \n\t  ", // Whitespace only
		},
	}

	err := worker.handle(ctx, job)

	// Should succeed without calling embed or index
	assert.NoError(t, err)
	mockEmbed.AssertNotCalled(t, "Embed")
	mockIndex.AssertNotCalled(t, "UpsertContext")
}

func TestHandleContextWithValidText(t *testing.T) {
	worker, mockEmbed, mockIndex := createTestWorker()
	ctx := context.Background()

	expectedVector := make([]float32, 768)

	job := job{
		id:          17,
		op:          OpUpsertContext,
		aggregateID: "test-context-id",
		payload: map[string]interface{}{
			"context": "Valid context text",
		},
	}

	mockEmbed.On("Embed", ctx, "Valid context text").Return(expectedVector, nil)
	mockIndex.On("UpsertContext", ctx, "test-context-id", expectedVector, job.payload).Return(nil)

	err := worker.handle(ctx, job)

	assert.NoError(t, err)
	mockEmbed.AssertCalled(t, "Embed", ctx, "Valid context text")
	mockIndex.AssertCalled(t, "UpsertContext", ctx, "test-context-id", expectedVector, job.payload)
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestIsAlreadyExists(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"contains already exists", errors.New("object already exists"), true},
		{"contains status 422", errors.New("status code: 422"), true},
		{"unrelated error", errors.New("connection refused"), false},
		{"mixed case", errors.New("Already Exists in index"), false}, // strings.Contains is case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAlreadyExists(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreferredText(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]interface{}
		keys     []string
		expected string
	}{
		{
			name:     "prefers first non-empty",
			payload:  map[string]interface{}{"a": "", "b": "value", "c": "other"},
			keys:     []string{"a", "b", "c"},
			expected: "value",
		},
		{
			name:     "all empty returns empty",
			payload:  map[string]interface{}{"a": "", "b": "", "c": ""},
			keys:     []string{"a", "b", "c"},
			expected: "",
		},
		{
			name:     "missing fields returns empty",
			payload:  map[string]interface{}{"other": "value"},
			keys:     []string{"a", "b"},
			expected: "",
		},
		{
			name:     "non-string ignored",
			payload:  map[string]interface{}{"a": 123, "b": "text"},
			keys:     []string{"a", "b"},
			expected: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preferredText(tt.payload, tt.keys...)
			assert.Equal(t, tt.expected, result)
		})
	}
}
