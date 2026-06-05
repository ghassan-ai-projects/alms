package store

import (
	"testing"
)

func TestAgentStoreSkipNoPG(t *testing.T) {
	t.Skip("PostgreSQL required — skipping agent store tests")
}

func TestAgentStoreCRUD(t *testing.T) {
	t.Skip("PostgreSQL required — skipping agent store CRUD tests")
}

func TestAgentStoreSyncAckValidation(t *testing.T) {
	t.Skip("PostgreSQL required — skipping agent store sync ack tests")
}
