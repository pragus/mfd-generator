//nolint:all
package runtime

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

// Applier is the shared go-pg query transformation contract.
type Applier = func(query *orm.Query) (*orm.Query, error)

// RepoOption configures generated generic repositories.
type RepoOption func(*repoConfig)

type repoConfig struct {
	filters        []Applier
	insertDefaults []Applier
	updateDefaults []Applier
	deleteOptions  []Applier
	setDeleted     func(any)
}

// WithFilters adds filters applied to every read query.
func WithFilters(filters ...Applier) RepoOption {
	return func(config *repoConfig) {
		config.filters = append(config.filters, filters...)
	}
}

// WithInsertDefaults configures columns applied when no insert options are supplied.
func WithInsertDefaults(options ...Applier) RepoOption {
	return func(config *repoConfig) {
		config.insertDefaults = append(config.insertDefaults, options...)
	}
}

// WithUpdateDefaults configures columns applied when no update options are supplied.
func WithUpdateDefaults(options ...Applier) RepoOption {
	return func(config *repoConfig) {
		config.updateDefaults = append(config.updateDefaults, options...)
	}
}

// WithSoftDelete configures status-based deletion for a generated model.
func WithSoftDelete[T any](setDeleted func(*T), options ...Applier) RepoOption {
	return func(config *repoConfig) {
		config.setDeleted = func(model any) {
			setDeleted(model.(*T))
		}
		config.deleteOptions = append(config.deleteOptions, options...)
	}
}

// GenRepo provides generic repository operations for one generated model.
type GenRepo[T any] struct {
	db             orm.DB
	filters        []Applier
	insertDefaults []Applier
	updateDefaults []Applier
	deleteOptions  []Applier
	setDeleted     func(any)
}

// NewGenRepo creates a generic repository backed by go-pg/v10.
func NewGenRepo[T any](db orm.DB, options ...RepoOption) GenRepo[T] {
	config := repoConfig{}
	for _, option := range options {
		option(&config)
	}

	return GenRepo[T]{
		db:             db,
		filters:        cloneAppliers(config.filters),
		insertDefaults: cloneAppliers(config.insertDefaults),
		updateDefaults: cloneAppliers(config.updateDefaults),
		deleteOptions:  cloneAppliers(config.deleteOptions),
		setDeleted:     config.setDeleted,
	}
}

// WithTransaction returns a repository bound to tx.
func (repo GenRepo[T]) WithTransaction(tx *pg.Tx) GenRepo[T] {
	repo.db = tx
	return repo
}

// AppendFilter returns a repository with one additional base filter.
func (repo GenRepo[T]) AppendFilter(filter Applier) GenRepo[T] {
	repo.filters = append(cloneAppliers(repo.filters), filter)
	return repo
}

// One returns one model or nil when no row matches.
func (repo GenRepo[T]) One(ctx context.Context, search Applier, options ...Applier) (*T, error) {
	model := new(T)
	query, err := repo.readQuery(ctx, model, search, nil, options...)
	if err != nil {
		return nil, err
	}
	err = query.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return model, nil
}

// List returns all models matching the query.
func (repo GenRepo[T]) List(ctx context.Context, search, pager Applier, options ...Applier) ([]T, error) {
	models := make([]T, 0)
	query, err := repo.readQuery(ctx, &models, search, pager, options...)
	if err != nil {
		return nil, err
	}
	if err := query.Select(); err != nil {
		return nil, err
	}
	return models, nil
}

// Count returns the number of matching models.
func (repo GenRepo[T]) Count(ctx context.Context, search Applier, options ...Applier) (int, error) {
	model := new(T)
	query, err := repo.readQuery(ctx, model, search, nil, options...)
	if err != nil {
		return 0, err
	}
	return query.Count()
}

// Add inserts model and returns the same pointer.
func (repo GenRepo[T]) Add(ctx context.Context, model *T, options ...Applier) (*T, error) {
	query := repo.db.ModelContext(ctx, model)
	if len(options) == 0 {
		options = repo.insertDefaults
	}
	query, err := applyAppliers(query, options...)
	if err != nil {
		return nil, err
	}
	_, err = query.Insert()
	if err != nil {
		return nil, err
	}
	return model, nil
}

// Update updates model by its primary key and reports whether a row changed.
func (repo GenRepo[T]) Update(ctx context.Context, model *T, options ...Applier) (bool, error) {
	query := repo.db.ModelContext(ctx, model).WherePK()
	if len(options) == 0 {
		options = repo.updateDefaults
	}
	query, err := applyAppliers(query, options...)
	if err != nil {
		return false, err
	}
	result, err := query.Update()
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

// Delete deletes model by its primary key and reports whether a row changed.
func (repo GenRepo[T]) Delete(ctx context.Context, model *T, options ...Applier) (bool, error) {
	if repo.setDeleted != nil {
		repo.setDeleted(model)
		options = append(cloneAppliers(options), repo.deleteOptions...)
		query, err := applyAppliers(repo.db.ModelContext(ctx, model).WherePK(), options...)
		if err != nil {
			return false, err
		}
		result, err := query.Update()
		if err != nil {
			return false, err
		}
		return result.RowsAffected() > 0, nil
	}

	query, err := applyAppliers(repo.db.ModelContext(ctx, model).WherePK(), options...)
	if err != nil {
		return false, err
	}
	result, err := query.Delete()
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (repo GenRepo[T]) readQuery(ctx context.Context, model any, search, pager Applier, options ...Applier) (*orm.Query, error) {
	query := repo.db.ModelContext(ctx, model)
	var err error
	if query, err = applyAppliers(query, repo.filters...); err != nil {
		return nil, err
	}
	if query, err = applyAppliers(query, search); err != nil {
		return nil, err
	}
	if query, err = applyAppliers(query, pager); err != nil {
		return nil, err
	}
	return applyAppliers(query, options...)
}

func applyAppliers(query *orm.Query, appliers ...Applier) (*orm.Query, error) {
	for _, applier := range appliers {
		if applier != nil {
			var err error
			query, err = applier(query)
			if err != nil {
				return query, err
			}
		}
	}
	return query, nil
}

func cloneAppliers(appliers []Applier) []Applier {
	return append([]Applier(nil), appliers...)
}
