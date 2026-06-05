package store

import (
	"testing"
)

func TestProtocolStoreSkipNoPG(t *testing.T) {
	t.Skip("PostgreSQL required — skipping protocol store tests")
}

func TestProtocolStoreCRUD(t *testing.T) {
	t.Skip("PostgreSQL required — skipping protocol store CRUD tests")
}

func TestProtocolStorePull(t *testing.T) {
	t.Skip("PostgreSQL required — skipping protocol store pull tests")
}
