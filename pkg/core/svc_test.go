package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	repo := NewMockUserRepo(t)
	prov := NewMockMITProv(t)

	svc := New(repo, prov)

	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Equal(t, prov, svc.prov)
}
