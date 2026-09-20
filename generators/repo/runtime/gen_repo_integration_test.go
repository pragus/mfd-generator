//go:build integration

package runtime

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type repoTestCategory struct {
	tableName   struct{}        `pg:"categories,alias:t,discard_unknown_columns"`
	ID          int             `pg:"categoryId,pk"`
	Title       string          `pg:"title,use_zero"`
	OrderNumber int             `pg:"orderNumber,use_zero"`
	StatusID    int             `pg:"statusId,use_zero"`
	Status      *repoTestStatus `pg:"rel:has-one,join_fk:statusId"`
}

type repoTestStatus struct {
	tableName struct{} `pg:"statuses,alias:s,discard_unknown_columns"`
	ID        int      `pg:"statusId,pk"`
}

func TestGenRepoCRUD(t *testing.T) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Fatal("DB_DSN is required")
	}
	options, err := pg.ParseURL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	db := pg.Connect(options)
	defer db.Close()

	ctx := context.Background()
	if _, err := db.Exec(`TRUNCATE "categories", "statuses" RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO "statuses" ("statusId") VALUES (1), (2), (3)`); err != nil {
		t.Fatal(err)
	}

	repo := NewGenRepo[repoTestCategory](db)
	category := &repoTestCategory{Title: "runtime", OrderNumber: 1, StatusID: 1}
	added, err := repo.Add(ctx, category)
	if err != nil || added != category || category.ID == 0 {
		t.Fatalf("Add() = %#v, %v", added, err)
	}

	found, err := repo.One(ctx, nil)
	if err != nil || found == nil || found.ID != category.ID {
		t.Fatalf("One() = %#v, %v", found, err)
	}
	list, err := repo.List(ctx, nil, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("List() = %d, %v", len(list), err)
	}
	count, err := repo.Count(ctx, nil)
	if err != nil || count != 1 {
		t.Fatalf("Count() = %d, %v", count, err)
	}

	second, err := repo.Add(ctx, &repoTestCategory{Title: "second", OrderNumber: 2, StatusID: 1})
	if err != nil {
		t.Fatal(err)
	}
	orderByID := func(query *orm.Query) (*orm.Query, error) {
		return query.OrderExpr(`"categoryId" ASC`), nil
	}
	var ids []int
	if err := repo.ForEach(ctx, nil, orderByID, func(value repoTestCategory) error {
		ids = append(ids, value.ID)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != category.ID || ids[1] != second.ID {
		t.Fatalf("ForEach() IDs = %v, want [%d %d]", ids, category.ID, second.ID)
	}

	visitErr := errors.New("stop visiting")
	calls := 0
	err = repo.ForEach(ctx, nil, orderByID, func(repoTestCategory) error {
		calls++
		return visitErr
	})
	if !errors.Is(err, visitErr) {
		t.Fatalf("ForEach() error = %v, want %v", err, visitErr)
	}
	if calls != 1 {
		t.Fatalf("ForEach() calls = %d, want 1", calls)
	}
	withStatus := func(query *orm.Query) (*orm.Query, error) {
		return query.Relation("Status"), nil
	}
	if err := repo.ForEach(ctx, withStatus, nil, func(repoTestCategory) error { return nil }); !errors.Is(err, ErrForEachRelations) {
		t.Fatalf("ForEach() relation error = %v", err)
	}
	if _, err := repo.Delete(ctx, second); err != nil {
		t.Fatal(err)
	}

	category.Title = "updated"
	updated, err := repo.Update(ctx, category)
	if err != nil || !updated {
		t.Fatalf("Update() = %v, %v", updated, err)
	}
	deleted, err := repo.Delete(ctx, category)
	if err != nil || !deleted {
		t.Fatalf("Delete() = %v, %v", deleted, err)
	}

	softRepo := NewGenRepo[repoTestCategory](db, WithSoftDelete(func(model *repoTestCategory) {
		model.StatusID = 3
	}))
	softCategory := &repoTestCategory{Title: "soft", OrderNumber: 2, StatusID: 1}
	if _, err := softRepo.Add(ctx, softCategory); err != nil {
		t.Fatal(err)
	}
	softDeleted, err := softRepo.Delete(ctx, softCategory)
	if err != nil || !softDeleted || softCategory.StatusID != 3 {
		t.Fatalf("soft Delete() = %v, %v; model = %#v", softDeleted, err, softCategory)
	}

	rollback := errors.New("rollback")
	err = db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		txRepo := NewGenRepo[repoTestCategory](db).WithTransaction(tx)
		if _, err := txRepo.Add(ctx, &repoTestCategory{Title: "rolled back", OrderNumber: 3, StatusID: 1}); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("RunInTransaction() error = %v", err)
	}
	count, err = repo.Count(ctx, func(query *orm.Query) (*orm.Query, error) {
		return query.Where(`"title" = ?`, "rolled back"), nil
	})
	if err != nil || count != 0 {
		t.Fatalf("Count() after rollback = %d, %v", count, err)
	}
}
