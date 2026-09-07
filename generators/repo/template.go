package repo

const repoDefaultTemplate = `
package {{.Package}}

import (
	"context"
	"errors"{{if .HasImports}}{{range .Imports}}
	"{{.}}"{{end}}
	{{end}}

	"github.com/go-pg/pg{{.GoPGVer}}"
	"github.com/go-pg/pg{{.GoPGVer}}/orm"
)

type {{.Name}}Repo struct {
	db orm.DB
	filters map[string][]Filter
	sort    map[string][]SortField
	join    map[string][]string
}

// New{{.Name}}Repo returns new repository
func New{{.Name}}Repo(db orm.DB) {{.Name}}Repo {
	return {{.Name}}Repo{
		db:     db,
		filters: map[string][]Filter{
			{{- range .Entities}}{{if .HasStatus}}
			Tables.{{.Name}}.Name: {StatusFilter}, {{end}}{{end}} 
		},
		sort: map[string][]SortField{
			{{- range .Entities}}{{if ne .SortField ""}}
			Tables.{{.Name}}.Name: { {Column: Columns.{{.Name}}.{{.SortField}}, Direction: {{.SortDir}}} },{{end}}{{end}}
		},
		join: map[string][]string{
			{{- range $i, $e := .Entities}}
			Tables.{{$e.Name}}.Name: {TableColumns{{range .Relations}}, Columns.{{$e.Name}}.{{.Name}}{{end}} },{{end}} 
		},
	}
}

// WithTransaction is a function that wraps {{.Name}}Repo with pg.Tx transaction.
func ({{.ShortVarName}}r {{.Name}}Repo) WithTransaction(tx *pg.Tx) {{.Name}}Repo {
	{{.ShortVarName}}r.db = tx
	return {{.ShortVarName}}r
}

// WithEnabledOnly is a function that adds "statusId"=1 as base filter.
func ({{.ShortVarName}}r {{.Name}}Repo) WithEnabledOnly() {{.Name}}Repo {
	f := make(map[string][]Filter,len({{.ShortVarName}}r.filters))
	for i := range {{.ShortVarName}}r.filters {
    	f[i] = make([]Filter,len({{.ShortVarName}}r.filters[i]))
        copy(f[i], {{.ShortVarName}}r.filters[i])
        f[i] = append(f[i], StatusEnabledFilter)
	}
	{{.ShortVarName}}r.filters = f

	return {{.ShortVarName}}r
}

{{range $i, $e := .Entities}}/*** {{.Name}} ***/

// Full{{.Name}} returns full joins with all columns
func ({{$.ShortVarName}}r {{$.Name}}Repo) Full{{.Name}}() OpFunc {
	return WithColumns({{$.ShortVarName}}r.join[Tables.{{.Name}}.Name]...)
}

// Default{{.Name}}Sort returns default sort.
func ({{$.ShortVarName}}r {{$.Name}}Repo) Default{{.Name}}Sort() OpFunc {
	return WithSort({{$.ShortVarName}}r.sort[Tables.{{.Name}}.Name]...)
}
{{if .HasPKs}}
// {{.Name}}ByID is a function that returns {{.Name}} by ID(s) or nil.
func ({{$.ShortVarName}}r {{$.Name}}Repo) {{.Name}}ByID(ctx context.Context{{range .PKs}}, {{.Arg}} {{.Type}}{{end}}, ops ...OpFunc) (*{{.Name}}, error) {
	return {{$.ShortVarName}}r.One{{.Name}}(ctx, &{{.Name}}Search{ {{range $i, $e := .PKs}}{{if $i}}, {{end}}{{.Field}}: &{{.Arg}}{{end}} }, ops...)
}
{{end}}

// One{{.Name}} is a function that returns one {{.Name}} by filters. It could return pg.ErrMultiRows.
func ({{$.ShortVarName}}r {{$.Name}}Repo) One{{.Name}}(ctx context.Context, search *{{.Name}}Search, ops ...OpFunc) (*{{.Name}}, error) {
	obj := &{{.Name}}{}
	err := buildQuery(ctx, {{$.ShortVarName}}r.db, obj, search, {{$.ShortVarName}}r.filters[Tables.{{.Name}}.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// {{.NamePlural}}ByFilters returns {{.Name}} list.
func ({{$.ShortVarName}}r {{$.Name}}Repo) {{.NamePlural}}ByFilters(ctx context.Context, search *{{.Name}}Search, pager Pager, ops ...OpFunc) ({{.VarNamePlural}} []{{.Name}}, err error) {
	err = buildQuery(ctx, {{$.ShortVarName}}r.db, &{{.VarNamePlural}}, search, {{$.ShortVarName}}r.filters[Tables.{{.Name}}.Name], pager, ops...).Select()
	return
}

// Count{{.NamePlural}} returns count
func ({{$.ShortVarName}}r {{$.Name}}Repo) Count{{.NamePlural}}(ctx context.Context, search *{{.Name}}Search, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, {{$.ShortVarName}}r.db, &{{.Name}}{}, search, {{$.ShortVarName}}r.filters[Tables.{{.Name}}.Name], PagerOne, ops...).Count()
}

// Add{{.Name}} adds {{.Name}} to DB.
func ({{$.ShortVarName}}r {{$.Name}}Repo) Add{{.Name}}(ctx context.Context, {{.VarName}} *{{.Name}}, ops ...OpFunc) (*{{.Name}}, error) {
	q := {{$.ShortVarName}}r.db.ModelContext(ctx, {{.VarName}})
	{{- if .HasNotAddable }}
	if len(ops) == 0 {
		q = q.ExcludeColumn({{range .NotAddable}}Columns.{{$e.Name}}.{{.}},{{end}})
	}
	{{- end }}
	applyOps(q, ops...)
	_, err := q.Insert()

	return {{.VarName}}, err
}

// Update{{.Name}} updates {{.Name}} in DB.
func ({{$.ShortVarName}}r {{$.Name}}Repo) Update{{.Name}}(ctx context.Context, {{.VarName}} *{{.Name}}, ops ...OpFunc) (bool, error) {
	q := {{$.ShortVarName}}r.db.ModelContext(ctx, {{.VarName}}).WherePK()
	{{- if .HasNotUpdatable }}
	if len(ops) == 0 {
		q = q.ExcludeColumn({{range .NotUpdatable}}Columns.{{$e.Name}}.{{.}},{{end}})
    }
    {{- end }}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}
{{if .HasPKs}}
// Delete{{.Name}} {{if .HasStatus}}set statusId to deleted in DB{{else}}deletes {{.Name}} from DB{{end}}.
func ({{$.ShortVarName}}r {{$.Name}}Repo) Delete{{.Name}}(ctx context.Context{{range .PKs}}, {{.Arg}} {{.Type}}{{end}}) (deleted bool, err error) {
	{{.VarName}} := &{{.Name}}{ {{range $i, $e := .PKs}}{{if $i}}, {{end}}{{.Field}}: {{.Arg}}{{end}}{{if .HasStatus}}, StatusID: StatusDeleted,{{end}} }

{{if .HasStatus}}return {{$.ShortVarName}}r.Update{{.Name}}(ctx, {{.VarName}}, WithColumns(Columns.{{.Name}}.StatusID)){{else}}res, err := {{$.ShortVarName}}r.db.ModelContext(ctx, {{.VarName}}).WherePK().Delete()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err{{end}}
}{{end}}
{{end}}`

const repoGenericTemplate = `package {{.Package}}

import (
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type {{.Name}}Repo struct {
{{- range .Entities}}
	{{.Name}} GenRepo[{{.Name}}]
{{- end}}
}

// New{{.Name}}Repo returns a generic repository collection.
func New{{.Name}}Repo(db orm.DB) {{.Name}}Repo {
return {{.Name}}Repo{
{{- range .Entities}}
{{- $entity := .}}
		{{$entity.Name}}: NewGenRepo[{{$entity.Name}}](db{{if $entity.HasStatus}}, WithFilters(ApplyFilter(StatusFilter)), WithSoftDelete(func(model *{{$entity.Name}}) { model.StatusID = StatusDeleted }, ApplyOp(WithColumns(Columns.{{$entity.Name}}.StatusID)), ApplyOp(func(query *orm.Query) { query.Set("? = ?", pg.Ident(Columns.{{$entity.Name}}.StatusID), StatusDeleted) })){{end}}{{if $entity.HasNotAddable}}, WithInsertDefaults(ApplyOp(WithoutColumns({{range $entity.NotAddable}}Columns.{{$entity.Name}}.{{.}},{{end}}))){{end}}{{if $entity.HasNotUpdatable}}, WithUpdateDefaults(ApplyOp(WithoutColumns({{range $entity.NotUpdatable}}Columns.{{$entity.Name}}.{{.}},{{end}}))){{end}}),
{{- end}}
	}
}

// WithTransaction returns a repository collection bound to tx.
func (repo {{.Name}}Repo) WithTransaction(tx *pg.Tx) {{.Name}}Repo {
{{- range .Entities}}
	repo.{{.Name}} = repo.{{.Name}}.WithTransaction(tx)
{{- end}}
	return repo
}

// WithEnabledOnly returns a repository collection that selects enabled rows.
func (repo {{.Name}}Repo) WithEnabledOnly() {{.Name}}Repo {
{{- range .Entities}}{{if .HasStatus}}
	repo.{{.Name}} = repo.{{.Name}}.AppendFilter(ApplyFilter(StatusEnabledFilter))
{{- end}}{{- end}}
	return repo
}

{{range .Entities}}
{{- $entity := .}}
// Full{{.Name}} returns an operation that loads all generated relations.
func (repo {{$.Name}}Repo) Full{{.Name}}() Applier {
	return ApplyOp(WithColumns(TableColumns{{range $entity.Relations}}, Columns.{{$entity.Name}}.{{.Name}}{{end}}))
}

// Default{{.Name}}Sort returns an operation for the generated default sort.
func (repo {{$.Name}}Repo) Default{{.Name}}Sort() Applier {
{{- if ne .SortField ""}}
	return ApplyOp(WithSort(SortField{Column: Columns.{{.Name}}.{{.SortField}}, Direction: {{.SortDir}}}))
{{- else}}
	return func(query *orm.Query) (*orm.Query, error) { return query, nil }
{{- end}}
}

// {{.Name}}Query returns typed search and pager appliers.
func (repo {{$.Name}}Repo) {{.Name}}Query(search *{{.Name}}Search, pager Pager) (Applier, Applier) {
	var searchApply Applier
	if search != nil {
		searchApply = search.Q()
	}
	return searchApply, ApplyPager(pager)
}
{{end}}
`
