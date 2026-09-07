package runtime

import (
	"errors"
	"testing"

	"github.com/go-pg/pg/v10/orm"
)

func TestApplyAppliersStopsAtFirstError(t *testing.T) {
	query := new(orm.Query)
	sentinel := errors.New("sentinel applier error")
	called := false

	got, err := applyAppliers(query,
		func(query *orm.Query) (*orm.Query, error) {
			return query, sentinel
		},
		func(query *orm.Query) (*orm.Query, error) {
			called = true
			return query, nil
		},
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf("applyAppliers() error = %v, want %v", err, sentinel)
	}
	if got != query {
		t.Fatalf("applyAppliers() query = %p, want %p", got, query)
	}
	if called {
		t.Fatal("applyAppliers() continued after an error")
	}
}

func TestNewGenRepoCopiesOptionAppliers(t *testing.T) {
	query := new(orm.Query)
	firstError := errors.New("first")
	first := func(query *orm.Query) (*orm.Query, error) { return query, firstError }
	second := func(query *orm.Query) (*orm.Query, error) {
		return query, errors.New("second")
	}
	filters := []Applier{first}

	repo := NewGenRepo[struct{}](nil, WithFilters(filters...))
	filters[0] = second

	_, err := repo.filters[0](query)
	if !errors.Is(err, firstError) {
		t.Fatalf("NewGenRepo() copied applier error = %v, want %v", err, firstError)
	}
}
