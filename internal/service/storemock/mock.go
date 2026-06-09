// Package storemock provides mock store implementations for service-level tests.
package storemock

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// AgentStore is an in-memory mock of service.AgentStore.
type AgentStore struct {
	mu     sync.Mutex
	agents map[string]*models.AgentSpec
	err    error // injected error for testing failure paths
}

// NewAgentStore creates an empty AgentStore mock.
func NewAgentStore() *AgentStore {
	return &AgentStore{
		agents: make(map[string]*models.AgentSpec),
	}
}

// SetError configures an error to be returned on all subsequent calls.
func (m *AgentStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *AgentStore) getErr() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

func (m *AgentStore) Create(ctx context.Context, spec models.AgentSpec) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.agents[spec.AgentID]; exists {
		return fmt.Errorf("%w: agent %s", models.ErrConflict, spec.AgentID)
	}
	cp := spec
	m.agents[spec.AgentID] = &cp
	return nil
}

func (m *AgentStore) Get(ctx context.Context, agentID string) (models.AgentSpec, error) {
	if err := m.getErr(); err != nil {
		return models.AgentSpec{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.agents[agentID]
	if !ok {
		return models.AgentSpec{}, fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}
	return *a, nil
}

func (m *AgentStore) Update(ctx context.Context, spec models.AgentSpec) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[spec.AgentID]; !ok {
		return fmt.Errorf("agent %s: %w", spec.AgentID, models.ErrNotFound)
	}
	cp := spec
	m.agents[spec.AgentID] = &cp
	return nil
}

func (m *AgentStore) Delete(ctx context.Context, agentID string) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[agentID]; !ok {
		return fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}
	delete(m.agents, agentID)
	return nil
}

func (m *AgentStore) Heartbeat(ctx context.Context, agentID string) (time.Time, error) {
	if err := m.getErr(); err != nil {
		return time.Time{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.agents[agentID]
	if !ok {
		return time.Time{}, fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}
	now := time.Now()
	a.LastHeartbeat = now
	return now, nil
}

func (m *AgentStore) List(ctx context.Context, filter map[string]string, limit, offset int) ([]models.AgentSpec, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []models.AgentSpec
	for _, a := range m.agents {
		if agentType, ok := filter["agent_type"]; ok && agentType != "" {
			if string(a.AgentType) != agentType {
				continue
			}
		}
		filtered = append(filtered, *a)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	if offset >= len(filtered) {
		return []models.AgentSpec{}, nil
	}
	filtered = filtered[offset:]
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	if filtered == nil {
		return []models.AgentSpec{}, nil
	}
	return filtered, nil
}

// Count returns the total number of registered agents.
func (m *AgentStore) Count(ctx context.Context) (int, error) {
	if err := m.getErr(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.agents), nil
}

// GetAll returns a copy of all stored agents (for test assertions).
func (m *AgentStore) GetAll() map[string]models.AgentSpec {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]models.AgentSpec, len(m.agents))
	for k, v := range m.agents {
		result[k] = *v
	}
	return result
}

// LearningStore is an in-memory mock of service.LearningStore.
type LearningStore struct {
	mu           sync.Mutex
	records      map[string]*models.LearningRecord
	acks         map[string][]string // agentID -> []learningID
	agentCursors map[string]time.Time
	err          error
}

// NewLearningStore creates an empty LearningStore mock.
func NewLearningStore() *LearningStore {
	return &LearningStore{
		records:      make(map[string]*models.LearningRecord),
		acks:         make(map[string][]string),
		agentCursors: make(map[string]time.Time),
	}
}

// SetError configures an error to be returned on all subsequent calls.
func (m *LearningStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *LearningStore) getErr() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

// Create stores a learning record and returns the ID.
func (m *LearningStore) Create(ctx context.Context, record models.LearningRecord) (string, error) {
	if err := m.getErr(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("lrn-%d", len(m.records)+1)
	cp := record
	cp.LearningID = id
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	cp.EnrichmentMetadata = models.NormalizeEnrichmentMetadata(cp.EnrichmentMetadata)
	m.records[id] = &cp
	return id, nil
}

// Get retrieves a learning record by ID.
func (m *LearningStore) Get(ctx context.Context, learningID string) (models.LearningRecord, error) {
	if err := m.getErr(); err != nil {
		return models.LearningRecord{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[learningID]
	if !ok {
		return models.LearningRecord{}, fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	return *rec, nil
}

// Sync returns learnings after the given timestamp, optionally filtered.
func (m *LearningStore) Sync(ctx context.Context, agentID string, since time.Time, ltype string, tags []string) ([]models.LearningRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.LearningRecord
	for _, rec := range m.records {
		if !rec.CreatedAt.After(since) {
			continue
		}
		if rec.IsDeleted {
			continue
		}
		if ltype != "" && string(rec.Type) != ltype {
			continue
		}
		if len(tags) > 0 {
			tagMatch := false
			for _, t := range tags {
				for _, rt := range rec.Tags {
					if t == rt {
						tagMatch = true
						break
					}
				}
				if tagMatch {
					break
				}
			}
			if !tagMatch {
				continue
			}
		}
		result = append(result, *rec)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

// SyncAck stores acknowledgements and advances the agent cursor.
func (m *LearningStore) SyncAck(ctx context.Context, agentID string, learningIDs []string) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.acks[agentID] = append(m.acks[agentID], learningIDs...)

	// Advance cursor to latest acknowledged learning's timestamp
	if len(learningIDs) > 0 {
		lastID := learningIDs[len(learningIDs)-1]
		if rec, ok := m.records[lastID]; ok {
			m.agentCursors[agentID] = rec.CreatedAt
		}
	}
	return nil
}

// Search performts search on stored records (simple substring match).
func (m *LearningStore) Search(ctx context.Context, query, ltype string, tags []string, limit int) ([]models.LearningRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.LearningRecord
	for _, rec := range m.records {
		if rec.IsDeleted {
			continue
		}
		// Simple substring matching
		if !containsSubstring(rec.Title, query) && !containsSubstring(rec.Body, query) {
			continue
		}
		if ltype != "" && string(rec.Type) != ltype {
			continue
		}
		if len(tags) > 0 {
			tagMatch := false
			for _, t := range tags {
				for _, rt := range rec.Tags {
					if t == rt {
						tagMatch = true
						break
					}
				}
				if tagMatch {
					break
				}
			}
			if !tagMatch {
				continue
			}
		}
		result = append(result, *rec)
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// SoftDelete marks a learning record as deleted.
func (m *LearningStore) SoftDelete(ctx context.Context, learningID string) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[learningID]
	if !ok {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	rec.IsDeleted = true
	now := time.Now()
	rec.DeletedAt = &now
	return nil
}

// ExpectedSyncIDs returns the ordered IDs of learnings after the given time.
func (m *LearningStore) ExpectedSyncIDs(ctx context.Context, agentID string, since time.Time) ([]string, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var ids []string
	var records []*models.LearningRecord
	for _, rec := range m.records {
		if rec.CreatedAt.After(since) && !rec.IsDeleted {
			records = append(records, rec)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
	for _, rec := range records {
		ids = append(ids, rec.LearningID)
	}
	return ids, nil
}

// GetAcks returns the acknowledged IDs for an agent (for test assertions).
func (m *LearningStore) GetAcks(agentID string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	acks := m.acks[agentID]
	result := make([]string, len(acks))
	copy(result, acks)
	return result
}

// Supersede updates the resolution of a learning record and sets superseded_by.
func (m *LearningStore) Supersede(ctx context.Context, oldID, newID string) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[oldID]
	if !ok {
		return fmt.Errorf("learning %s: %w", oldID, models.ErrNotFound)
	}
	rec.Resolution = models.ResolutionSuperseded
	rec.SupersededBy = newID
	return nil
}

// UpdateEnrichment updates the enrichment_metadata for a learning record.
func (m *LearningStore) UpdateEnrichment(ctx context.Context, learningID string, enrichmentJSON json.RawMessage) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[learningID]
	if !ok {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	rec.EnrichmentMetadata = enrichmentJSON
	return nil
}

// SearchWithStatus searches with additional filter params for status and includeRejected.
func (m *LearningStore) SearchWithStatus(ctx context.Context, query string, ltype string, tags []string, limit int, status string, includeRejected bool) ([]models.LearningRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.LearningRecord
	for _, rec := range m.records {
		if rec.IsDeleted {
			continue
		}
		if !containsSubstring(rec.Title, query) && !containsSubstring(rec.Body, query) {
			continue
		}
		if ltype != "" && string(rec.Type) != ltype {
			continue
		}
		if len(tags) > 0 {
			tagMatch := false
			for _, t := range tags {
				for _, rt := range rec.Tags {
					if t == rt {
						tagMatch = true
						break
					}
				}
				if tagMatch {
					break
				}
			}
			if !tagMatch {
				continue
			}
		}

		enrichment := models.NormalizeEnrichmentMetadata(rec.EnrichmentMetadata)
		var meta map[string]any
		if err := json.Unmarshal(enrichment, &meta); err != nil {
			return nil, fmt.Errorf("parse enrichment metadata: %w", err)
		}

		recStatus, _ := meta["status"].(string)
		if status != "" && recStatus != status {
			continue
		}

		if !includeRejected {
			if visible, ok := meta["is_visible"].(bool); ok && !visible {
				continue
			}
		}

		cp := *rec
		cp.EnrichmentMetadata = enrichment
		result = append(result, cp)
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// UpdateScore updates the score of a learning record.
func (m *LearningStore) UpdateScore(ctx context.Context, learningID string, score float64) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[learningID]
	if !ok {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	rec.Score = score
	return nil
}

// UpdateLearningRecord overwrites the stored learning record with the given one.
// Used in tests to modify fields like CreatedAt that aren't exposed via the interface.
func (m *LearningStore) UpdateLearningRecord(id string, record models.LearningRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := record
	cp.LearningID = id
	m.records[id] = &cp
}

// GetAgentCursor returns the agent's sync cursor (for test assertions).
func (m *LearningStore) GetAgentCursor(agentID string) time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.agentCursors[agentID]
}

// ProtocolStore is an in-memory mock of service.ProtocolStore.
type ProtocolStore struct {
	mu        sync.Mutex
	protocols map[string]*models.ProtocolRecord
	err       error
}

// NewProtocolStore creates an empty ProtocolStore mock.
func NewProtocolStore() *ProtocolStore {
	return &ProtocolStore{
		protocols: make(map[string]*models.ProtocolRecord),
	}
}

// SetError configures an error to be returned on all subsequent calls.
func (m *ProtocolStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *ProtocolStore) getErr() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

// Create stores a protocol record and returns the ID.
func (m *ProtocolStore) Create(ctx context.Context, record models.ProtocolRecord) (string, error) {
	if err := m.getErr(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("proto-%d", len(m.protocols)+1)
	cp := record
	cp.ProtocolID = id
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	m.protocols[id] = &cp
	return id, nil
}

// Get retrieves a protocol by ID.
func (m *ProtocolStore) Get(ctx context.Context, protocolID string) (models.ProtocolRecord, error) {
	if err := m.getErr(); err != nil {
		return models.ProtocolRecord{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.protocols[protocolID]
	if !ok {
		return models.ProtocolRecord{}, fmt.Errorf("protocol %s: %w", protocolID, models.ErrNotFound)
	}
	return *p, nil
}

// Pull returns active protocols matching the given tags.
func (m *ProtocolStore) Pull(ctx context.Context, agentTags []string) ([]models.ProtocolRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.ProtocolRecord
	for _, p := range m.protocols {
		if !p.IsActive {
			continue
		}
		if len(p.TargetTags) == 0 || tagsOverlap(p.TargetTags, agentTags) {
			result = append(result, *p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

// PullSince returns protocols matching tags created after the given protocol ID.
func (m *ProtocolStore) PullSince(ctx context.Context, agentTags []string, sinceID string) ([]models.ProtocolRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	if sinceID == "" {
		return m.Pull(ctx, agentTags)
	}

	m.mu.Lock()
	sinceProto, ok := m.protocols[sinceID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("since protocol %s: %w", sinceID, models.ErrNotFound)
	}
	sinceTime := sinceProto.CreatedAt

	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.ProtocolRecord
	for _, p := range m.protocols {
		if !p.IsActive {
			continue
		}
		if !p.CreatedAt.After(sinceTime) {
			continue
		}
		if len(p.TargetTags) == 0 || tagsOverlap(p.TargetTags, agentTags) {
			result = append(result, *p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

// List returns all protocol records.
func (m *ProtocolStore) List(ctx context.Context) ([]models.ProtocolRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.ProtocolRecord
	for _, p := range m.protocols {
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

// containsSubstring is a case-insensitive substring check.
func containsSubstring(s, substr string) bool {
	if substr == "" {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// tagsOverlap checks if any tag in a overlaps with any tag in b.
func tagsOverlap(a, b []string) bool {
	for _, ta := range a {
		for _, tb := range b {
			if ta == tb {
				return true
			}
		}
	}
	return false
}
