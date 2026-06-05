package store

import (
	"testing"
)

func TestLearningStoreSkipNoPG(t *testing.T) {
	t.Skip("PostgreSQL required — skipping learning store tests")
}

func TestLearningStoreCRUD(t *testing.T) {
	t.Skip("PostgreSQL required — skipping learning store CRUD tests")
}

func TestLearningStoreSyncAckValidation(t *testing.T) {
	t.Skip("PostgreSQL required — skipping learning store sync ack tests")
}

func TestLearningStoreSearch(t *testing.T) {
	t.Skip("PostgreSQL required — skipping learning store search tests")
}
