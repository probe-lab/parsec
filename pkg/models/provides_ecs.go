// Code generated by SQLBoiler 4.14.1 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/queries/qmhelper"
	"github.com/volatiletech/strmangle"
)

// Provide is an object representing the database table.
type Provide struct {
	ID          int         `boil:"id" json:"id" toml:"id" yaml:"id"`
	SchedulerID int         `boil:"scheduler_id" json:"scheduler_id" toml:"scheduler_id" yaml:"scheduler_id"`
	NodeID      int         `boil:"node_id" json:"node_id" toml:"node_id" yaml:"node_id"`
	RTSize      int         `boil:"rt_size" json:"rt_size" toml:"rt_size" yaml:"rt_size"`
	Duration    float64     `boil:"duration" json:"duration" toml:"duration" yaml:"duration"`
	Cid         string      `boil:"cid" json:"cid" toml:"cid" yaml:"cid"`
	Error       null.String `boil:"error" json:"error,omitempty" toml:"error" yaml:"error,omitempty"`
	CreatedAt   time.Time   `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`

	R *provideR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L provideL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var ProvideColumns = struct {
	ID          string
	SchedulerID string
	NodeID      string
	RTSize      string
	Duration    string
	Cid         string
	Error       string
	CreatedAt   string
}{
	ID:          "id",
	SchedulerID: "scheduler_id",
	NodeID:      "node_id",
	RTSize:      "rt_size",
	Duration:    "duration",
	Cid:         "cid",
	Error:       "error",
	CreatedAt:   "created_at",
}

var ProvideTableColumns = struct {
	ID          string
	SchedulerID string
	NodeID      string
	RTSize      string
	Duration    string
	Cid         string
	Error       string
	CreatedAt   string
}{
	ID:          "provides_ecs.id",
	SchedulerID: "provides_ecs.scheduler_id",
	NodeID:      "provides_ecs.node_id",
	RTSize:      "provides_ecs.rt_size",
	Duration:    "provides_ecs.duration",
	Cid:         "provides_ecs.cid",
	Error:       "provides_ecs.error",
	CreatedAt:   "provides_ecs.created_at",
}

// Generated where

type whereHelperfloat64 struct{ field string }

func (w whereHelperfloat64) EQ(x float64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.EQ, x) }
func (w whereHelperfloat64) NEQ(x float64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.NEQ, x)
}
func (w whereHelperfloat64) LT(x float64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.LT, x) }
func (w whereHelperfloat64) LTE(x float64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelperfloat64) GT(x float64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.GT, x) }
func (w whereHelperfloat64) GTE(x float64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}
func (w whereHelperfloat64) IN(slice []float64) qm.QueryMod {
	values := make([]interface{}, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return qm.WhereIn(fmt.Sprintf("%s IN ?", w.field), values...)
}
func (w whereHelperfloat64) NIN(slice []float64) qm.QueryMod {
	values := make([]interface{}, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return qm.WhereNotIn(fmt.Sprintf("%s NOT IN ?", w.field), values...)
}

type whereHelpernull_String struct{ field string }

func (w whereHelpernull_String) EQ(x null.String) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, false, x)
}
func (w whereHelpernull_String) NEQ(x null.String) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, true, x)
}
func (w whereHelpernull_String) LT(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpernull_String) LTE(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpernull_String) GT(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpernull_String) GTE(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}
func (w whereHelpernull_String) IN(slice []string) qm.QueryMod {
	values := make([]interface{}, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return qm.WhereIn(fmt.Sprintf("%s IN ?", w.field), values...)
}
func (w whereHelpernull_String) NIN(slice []string) qm.QueryMod {
	values := make([]interface{}, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return qm.WhereNotIn(fmt.Sprintf("%s NOT IN ?", w.field), values...)
}

func (w whereHelpernull_String) IsNull() qm.QueryMod    { return qmhelper.WhereIsNull(w.field) }
func (w whereHelpernull_String) IsNotNull() qm.QueryMod { return qmhelper.WhereIsNotNull(w.field) }

var ProvideWhere = struct {
	ID          whereHelperint
	SchedulerID whereHelperint
	NodeID      whereHelperint
	RTSize      whereHelperint
	Duration    whereHelperfloat64
	Cid         whereHelperstring
	Error       whereHelpernull_String
	CreatedAt   whereHelpertime_Time
}{
	ID:          whereHelperint{field: "\"provides_ecs\".\"id\""},
	SchedulerID: whereHelperint{field: "\"provides_ecs\".\"scheduler_id\""},
	NodeID:      whereHelperint{field: "\"provides_ecs\".\"node_id\""},
	RTSize:      whereHelperint{field: "\"provides_ecs\".\"rt_size\""},
	Duration:    whereHelperfloat64{field: "\"provides_ecs\".\"duration\""},
	Cid:         whereHelperstring{field: "\"provides_ecs\".\"cid\""},
	Error:       whereHelpernull_String{field: "\"provides_ecs\".\"error\""},
	CreatedAt:   whereHelpertime_Time{field: "\"provides_ecs\".\"created_at\""},
}

// ProvideRels is where relationship names are stored.
var ProvideRels = struct {
	Node      string
	Scheduler string
}{
	Node:      "Node",
	Scheduler: "Scheduler",
}

// provideR is where relationships are stored.
type provideR struct {
	Node      *Node      `boil:"Node" json:"Node" toml:"Node" yaml:"Node"`
	Scheduler *Scheduler `boil:"Scheduler" json:"Scheduler" toml:"Scheduler" yaml:"Scheduler"`
}

// NewStruct creates a new relationship struct
func (*provideR) NewStruct() *provideR {
	return &provideR{}
}

func (r *provideR) GetNode() *Node {
	if r == nil {
		return nil
	}
	return r.Node
}

func (r *provideR) GetScheduler() *Scheduler {
	if r == nil {
		return nil
	}
	return r.Scheduler
}

// provideL is where Load methods for each relationship are stored.
type provideL struct{}

var (
	provideAllColumns            = []string{"id", "scheduler_id", "node_id", "rt_size", "duration", "cid", "error", "created_at"}
	provideColumnsWithoutDefault = []string{"scheduler_id", "node_id", "rt_size", "duration", "cid", "created_at"}
	provideColumnsWithDefault    = []string{"id", "error"}
	providePrimaryKeyColumns     = []string{"id"}
	provideGeneratedColumns      = []string{"id"}
)

type (
	// ProvideSlice is an alias for a slice of pointers to Provide.
	// This should almost always be used instead of []Provide.
	ProvideSlice []*Provide
	// ProvideHook is the signature for custom Provide hook methods
	ProvideHook func(context.Context, boil.ContextExecutor, *Provide) error

	provideQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	provideType                 = reflect.TypeOf(&Provide{})
	provideMapping              = queries.MakeStructMapping(provideType)
	providePrimaryKeyMapping, _ = queries.BindMapping(provideType, provideMapping, providePrimaryKeyColumns)
	provideInsertCacheMut       sync.RWMutex
	provideInsertCache          = make(map[string]insertCache)
	provideUpdateCacheMut       sync.RWMutex
	provideUpdateCache          = make(map[string]updateCache)
	provideUpsertCacheMut       sync.RWMutex
	provideUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

var provideAfterSelectHooks []ProvideHook

var provideBeforeInsertHooks []ProvideHook
var provideAfterInsertHooks []ProvideHook

var provideBeforeUpdateHooks []ProvideHook
var provideAfterUpdateHooks []ProvideHook

var provideBeforeDeleteHooks []ProvideHook
var provideAfterDeleteHooks []ProvideHook

var provideBeforeUpsertHooks []ProvideHook
var provideAfterUpsertHooks []ProvideHook

// doAfterSelectHooks executes all "after Select" hooks.
func (o *Provide) doAfterSelectHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideAfterSelectHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeInsertHooks executes all "before insert" hooks.
func (o *Provide) doBeforeInsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideBeforeInsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterInsertHooks executes all "after Insert" hooks.
func (o *Provide) doAfterInsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideAfterInsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeUpdateHooks executes all "before Update" hooks.
func (o *Provide) doBeforeUpdateHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideBeforeUpdateHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterUpdateHooks executes all "after Update" hooks.
func (o *Provide) doAfterUpdateHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideAfterUpdateHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeDeleteHooks executes all "before Delete" hooks.
func (o *Provide) doBeforeDeleteHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideBeforeDeleteHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterDeleteHooks executes all "after Delete" hooks.
func (o *Provide) doAfterDeleteHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideAfterDeleteHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeUpsertHooks executes all "before Upsert" hooks.
func (o *Provide) doBeforeUpsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideBeforeUpsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterUpsertHooks executes all "after Upsert" hooks.
func (o *Provide) doAfterUpsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range provideAfterUpsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// AddProvideHook registers your hook function for all future operations.
func AddProvideHook(hookPoint boil.HookPoint, provideHook ProvideHook) {
	switch hookPoint {
	case boil.AfterSelectHook:
		provideAfterSelectHooks = append(provideAfterSelectHooks, provideHook)
	case boil.BeforeInsertHook:
		provideBeforeInsertHooks = append(provideBeforeInsertHooks, provideHook)
	case boil.AfterInsertHook:
		provideAfterInsertHooks = append(provideAfterInsertHooks, provideHook)
	case boil.BeforeUpdateHook:
		provideBeforeUpdateHooks = append(provideBeforeUpdateHooks, provideHook)
	case boil.AfterUpdateHook:
		provideAfterUpdateHooks = append(provideAfterUpdateHooks, provideHook)
	case boil.BeforeDeleteHook:
		provideBeforeDeleteHooks = append(provideBeforeDeleteHooks, provideHook)
	case boil.AfterDeleteHook:
		provideAfterDeleteHooks = append(provideAfterDeleteHooks, provideHook)
	case boil.BeforeUpsertHook:
		provideBeforeUpsertHooks = append(provideBeforeUpsertHooks, provideHook)
	case boil.AfterUpsertHook:
		provideAfterUpsertHooks = append(provideAfterUpsertHooks, provideHook)
	}
}

// One returns a single provide record from the query.
func (q provideQuery) One(ctx context.Context, exec boil.ContextExecutor) (*Provide, error) {
	o := &Provide{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(ctx, exec, o)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for provides_ecs")
	}

	if err := o.doAfterSelectHooks(ctx, exec); err != nil {
		return o, err
	}

	return o, nil
}

// All returns all Provide records from the query.
func (q provideQuery) All(ctx context.Context, exec boil.ContextExecutor) (ProvideSlice, error) {
	var o []*Provide

	err := q.Bind(ctx, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to Provide slice")
	}

	if len(provideAfterSelectHooks) != 0 {
		for _, obj := range o {
			if err := obj.doAfterSelectHooks(ctx, exec); err != nil {
				return o, err
			}
		}
	}

	return o, nil
}

// Count returns the count of all Provide records in the query.
func (q provideQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count provides_ecs rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q provideQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if provides_ecs exists")
	}

	return count > 0, nil
}

// Node pointed to by the foreign key.
func (o *Provide) Node(mods ...qm.QueryMod) nodeQuery {
	queryMods := []qm.QueryMod{
		qm.Where("\"id\" = ?", o.NodeID),
	}

	queryMods = append(queryMods, mods...)

	return Nodes(queryMods...)
}

// Scheduler pointed to by the foreign key.
func (o *Provide) Scheduler(mods ...qm.QueryMod) schedulerQuery {
	queryMods := []qm.QueryMod{
		qm.Where("\"id\" = ?", o.SchedulerID),
	}

	queryMods = append(queryMods, mods...)

	return Schedulers(queryMods...)
}

// LoadNode allows an eager lookup of values, cached into the
// loaded structs of the objects. This is for an N-1 relationship.
func (provideL) LoadNode(ctx context.Context, e boil.ContextExecutor, singular bool, maybeProvide interface{}, mods queries.Applicator) error {
	var slice []*Provide
	var object *Provide

	if singular {
		var ok bool
		object, ok = maybeProvide.(*Provide)
		if !ok {
			object = new(Provide)
			ok = queries.SetFromEmbeddedStruct(&object, &maybeProvide)
			if !ok {
				return errors.New(fmt.Sprintf("failed to set %T from embedded struct %T", object, maybeProvide))
			}
		}
	} else {
		s, ok := maybeProvide.(*[]*Provide)
		if ok {
			slice = *s
		} else {
			ok = queries.SetFromEmbeddedStruct(&slice, maybeProvide)
			if !ok {
				return errors.New(fmt.Sprintf("failed to set %T from embedded struct %T", slice, maybeProvide))
			}
		}
	}

	args := make([]interface{}, 0, 1)
	if singular {
		if object.R == nil {
			object.R = &provideR{}
		}
		args = append(args, object.NodeID)

	} else {
	Outer:
		for _, obj := range slice {
			if obj.R == nil {
				obj.R = &provideR{}
			}

			for _, a := range args {
				if a == obj.NodeID {
					continue Outer
				}
			}

			args = append(args, obj.NodeID)

		}
	}

	if len(args) == 0 {
		return nil
	}

	query := NewQuery(
		qm.From(`nodes_ecs`),
		qm.WhereIn(`nodes_ecs.id in ?`, args...),
	)
	if mods != nil {
		mods.Apply(query)
	}

	results, err := query.QueryContext(ctx, e)
	if err != nil {
		return errors.Wrap(err, "failed to eager load Node")
	}

	var resultSlice []*Node
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice Node")
	}

	if err = results.Close(); err != nil {
		return errors.Wrap(err, "failed to close results of eager load for nodes_ecs")
	}
	if err = results.Err(); err != nil {
		return errors.Wrap(err, "error occurred during iteration of eager loaded relations for nodes_ecs")
	}

	if len(nodeAfterSelectHooks) != 0 {
		for _, obj := range resultSlice {
			if err := obj.doAfterSelectHooks(ctx, e); err != nil {
				return err
			}
		}
	}

	if len(resultSlice) == 0 {
		return nil
	}

	if singular {
		foreign := resultSlice[0]
		object.R.Node = foreign
		if foreign.R == nil {
			foreign.R = &nodeR{}
		}
		foreign.R.NodeProvidesEcs = append(foreign.R.NodeProvidesEcs, object)
		return nil
	}

	for _, local := range slice {
		for _, foreign := range resultSlice {
			if local.NodeID == foreign.ID {
				local.R.Node = foreign
				if foreign.R == nil {
					foreign.R = &nodeR{}
				}
				foreign.R.NodeProvidesEcs = append(foreign.R.NodeProvidesEcs, local)
				break
			}
		}
	}

	return nil
}

// LoadScheduler allows an eager lookup of values, cached into the
// loaded structs of the objects. This is for an N-1 relationship.
func (provideL) LoadScheduler(ctx context.Context, e boil.ContextExecutor, singular bool, maybeProvide interface{}, mods queries.Applicator) error {
	var slice []*Provide
	var object *Provide

	if singular {
		var ok bool
		object, ok = maybeProvide.(*Provide)
		if !ok {
			object = new(Provide)
			ok = queries.SetFromEmbeddedStruct(&object, &maybeProvide)
			if !ok {
				return errors.New(fmt.Sprintf("failed to set %T from embedded struct %T", object, maybeProvide))
			}
		}
	} else {
		s, ok := maybeProvide.(*[]*Provide)
		if ok {
			slice = *s
		} else {
			ok = queries.SetFromEmbeddedStruct(&slice, maybeProvide)
			if !ok {
				return errors.New(fmt.Sprintf("failed to set %T from embedded struct %T", slice, maybeProvide))
			}
		}
	}

	args := make([]interface{}, 0, 1)
	if singular {
		if object.R == nil {
			object.R = &provideR{}
		}
		args = append(args, object.SchedulerID)

	} else {
	Outer:
		for _, obj := range slice {
			if obj.R == nil {
				obj.R = &provideR{}
			}

			for _, a := range args {
				if a == obj.SchedulerID {
					continue Outer
				}
			}

			args = append(args, obj.SchedulerID)

		}
	}

	if len(args) == 0 {
		return nil
	}

	query := NewQuery(
		qm.From(`schedulers_ecs`),
		qm.WhereIn(`schedulers_ecs.id in ?`, args...),
	)
	if mods != nil {
		mods.Apply(query)
	}

	results, err := query.QueryContext(ctx, e)
	if err != nil {
		return errors.Wrap(err, "failed to eager load Scheduler")
	}

	var resultSlice []*Scheduler
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice Scheduler")
	}

	if err = results.Close(); err != nil {
		return errors.Wrap(err, "failed to close results of eager load for schedulers_ecs")
	}
	if err = results.Err(); err != nil {
		return errors.Wrap(err, "error occurred during iteration of eager loaded relations for schedulers_ecs")
	}

	if len(schedulerAfterSelectHooks) != 0 {
		for _, obj := range resultSlice {
			if err := obj.doAfterSelectHooks(ctx, e); err != nil {
				return err
			}
		}
	}

	if len(resultSlice) == 0 {
		return nil
	}

	if singular {
		foreign := resultSlice[0]
		object.R.Scheduler = foreign
		if foreign.R == nil {
			foreign.R = &schedulerR{}
		}
		foreign.R.SchedulerProvidesEcs = append(foreign.R.SchedulerProvidesEcs, object)
		return nil
	}

	for _, local := range slice {
		for _, foreign := range resultSlice {
			if local.SchedulerID == foreign.ID {
				local.R.Scheduler = foreign
				if foreign.R == nil {
					foreign.R = &schedulerR{}
				}
				foreign.R.SchedulerProvidesEcs = append(foreign.R.SchedulerProvidesEcs, local)
				break
			}
		}
	}

	return nil
}

// SetNode of the provide to the related item.
// Sets o.R.Node to related.
// Adds o to related.R.NodeProvidesEcs.
func (o *Provide) SetNode(ctx context.Context, exec boil.ContextExecutor, insert bool, related *Node) error {
	var err error
	if insert {
		if err = related.Insert(ctx, exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE \"provides_ecs\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, []string{"node_id"}),
		strmangle.WhereClause("\"", "\"", 2, providePrimaryKeyColumns),
	)
	values := []interface{}{related.ID, o.ID}

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, updateQuery)
		fmt.Fprintln(writer, values)
	}
	if _, err = exec.ExecContext(ctx, updateQuery, values...); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	o.NodeID = related.ID
	if o.R == nil {
		o.R = &provideR{
			Node: related,
		}
	} else {
		o.R.Node = related
	}

	if related.R == nil {
		related.R = &nodeR{
			NodeProvidesEcs: ProvideSlice{o},
		}
	} else {
		related.R.NodeProvidesEcs = append(related.R.NodeProvidesEcs, o)
	}

	return nil
}

// SetScheduler of the provide to the related item.
// Sets o.R.Scheduler to related.
// Adds o to related.R.SchedulerProvidesEcs.
func (o *Provide) SetScheduler(ctx context.Context, exec boil.ContextExecutor, insert bool, related *Scheduler) error {
	var err error
	if insert {
		if err = related.Insert(ctx, exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE \"provides_ecs\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, []string{"scheduler_id"}),
		strmangle.WhereClause("\"", "\"", 2, providePrimaryKeyColumns),
	)
	values := []interface{}{related.ID, o.ID}

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, updateQuery)
		fmt.Fprintln(writer, values)
	}
	if _, err = exec.ExecContext(ctx, updateQuery, values...); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	o.SchedulerID = related.ID
	if o.R == nil {
		o.R = &provideR{
			Scheduler: related,
		}
	} else {
		o.R.Scheduler = related
	}

	if related.R == nil {
		related.R = &schedulerR{
			SchedulerProvidesEcs: ProvideSlice{o},
		}
	} else {
		related.R.SchedulerProvidesEcs = append(related.R.SchedulerProvidesEcs, o)
	}

	return nil
}

// Provides retrieves all the records using an executor.
func Provides(mods ...qm.QueryMod) provideQuery {
	mods = append(mods, qm.From("\"provides_ecs\""))
	q := NewQuery(mods...)
	if len(queries.GetSelect(q)) == 0 {
		queries.SetSelect(q, []string{"\"provides_ecs\".*"})
	}

	return provideQuery{q}
}

// FindProvide retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindProvide(ctx context.Context, exec boil.ContextExecutor, iD int, selectCols ...string) (*Provide, error) {
	provideObj := &Provide{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"provides_ecs\" where \"id\"=$1", sel,
	)

	q := queries.Raw(query, iD)

	err := q.Bind(ctx, exec, provideObj)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from provides_ecs")
	}

	if err = provideObj.doAfterSelectHooks(ctx, exec); err != nil {
		return provideObj, err
	}

	return provideObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *Provide) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if o == nil {
		return errors.New("models: no provides_ecs provided for insertion")
	}

	var err error
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		if o.CreatedAt.IsZero() {
			o.CreatedAt = currTime
		}
	}

	if err := o.doBeforeInsertHooks(ctx, exec); err != nil {
		return err
	}

	nzDefaults := queries.NonZeroDefaultSet(provideColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	provideInsertCacheMut.RLock()
	cache, cached := provideInsertCache[key]
	provideInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			provideAllColumns,
			provideColumnsWithDefault,
			provideColumnsWithoutDefault,
			nzDefaults,
		)
		wl = strmangle.SetComplement(wl, provideGeneratedColumns)

		cache.valueMapping, err = queries.BindMapping(provideType, provideMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(provideType, provideMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"provides_ecs\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"provides_ecs\" %sDEFAULT VALUES%s"
		}

		var queryOutput, queryReturning string

		if len(cache.retMapping) != 0 {
			queryReturning = fmt.Sprintf(" RETURNING \"%s\"", strings.Join(returnColumns, "\",\""))
		}

		cache.query = fmt.Sprintf(cache.query, queryOutput, queryReturning)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}

	if err != nil {
		return errors.Wrap(err, "models: unable to insert into provides_ecs")
	}

	if !cached {
		provideInsertCacheMut.Lock()
		provideInsertCache[key] = cache
		provideInsertCacheMut.Unlock()
	}

	return o.doAfterInsertHooks(ctx, exec)
}

// Update uses an executor to update the Provide.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *Provide) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) (int64, error) {
	var err error
	if err = o.doBeforeUpdateHooks(ctx, exec); err != nil {
		return 0, err
	}
	key := makeCacheKey(columns, nil)
	provideUpdateCacheMut.RLock()
	cache, cached := provideUpdateCache[key]
	provideUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			provideAllColumns,
			providePrimaryKeyColumns,
		)
		wl = strmangle.SetComplement(wl, provideGeneratedColumns)

		if !columns.IsWhitelist() {
			wl = strmangle.SetComplement(wl, []string{"created_at"})
		}
		if len(wl) == 0 {
			return 0, errors.New("models: unable to update provides_ecs, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"provides_ecs\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, providePrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(provideType, provideMapping, append(wl, providePrimaryKeyColumns...))
		if err != nil {
			return 0, err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, values)
	}
	var result sql.Result
	result, err = exec.ExecContext(ctx, cache.query, values...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update provides_ecs row")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by update for provides_ecs")
	}

	if !cached {
		provideUpdateCacheMut.Lock()
		provideUpdateCache[key] = cache
		provideUpdateCacheMut.Unlock()
	}

	return rowsAff, o.doAfterUpdateHooks(ctx, exec)
}

// UpdateAll updates all rows with the specified column values.
func (q provideQuery) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) (int64, error) {
	queries.SetUpdate(q.Query, cols)

	result, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update all for provides_ecs")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to retrieve rows affected for provides_ecs")
	}

	return rowsAff, nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o ProvideSlice) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) (int64, error) {
	ln := int64(len(o))
	if ln == 0 {
		return 0, nil
	}

	if len(cols) == 0 {
		return 0, errors.New("models: update all requires at least one column argument")
	}

	colNames := make([]string, len(cols))
	args := make([]interface{}, len(cols))

	i := 0
	for name, value := range cols {
		colNames[i] = name
		args[i] = value
		i++
	}

	// Append all of the primary key values for each column
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), providePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"provides_ecs\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), len(colNames)+1, providePrimaryKeyColumns, len(o)))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update all in provide slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to retrieve rows affected all in update all provide")
	}
	return rowsAff, nil
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *Provide) Upsert(ctx context.Context, exec boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("models: no provides_ecs provided for upsert")
	}
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		if o.CreatedAt.IsZero() {
			o.CreatedAt = currTime
		}
	}

	if err := o.doBeforeUpsertHooks(ctx, exec); err != nil {
		return err
	}

	nzDefaults := queries.NonZeroDefaultSet(provideColumnsWithDefault, o)

	// Build cache key in-line uglily - mysql vs psql problems
	buf := strmangle.GetBuffer()
	if updateOnConflict {
		buf.WriteByte('t')
	} else {
		buf.WriteByte('f')
	}
	buf.WriteByte('.')
	for _, c := range conflictColumns {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(updateColumns.Kind))
	for _, c := range updateColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(insertColumns.Kind))
	for _, c := range insertColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzDefaults {
		buf.WriteString(c)
	}
	key := buf.String()
	strmangle.PutBuffer(buf)

	provideUpsertCacheMut.RLock()
	cache, cached := provideUpsertCache[key]
	provideUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			provideAllColumns,
			provideColumnsWithDefault,
			provideColumnsWithoutDefault,
			nzDefaults,
		)

		update := updateColumns.UpdateColumnSet(
			provideAllColumns,
			providePrimaryKeyColumns,
		)

		insert = strmangle.SetComplement(insert, provideGeneratedColumns)
		update = strmangle.SetComplement(update, provideGeneratedColumns)

		if updateOnConflict && len(update) == 0 {
			return errors.New("models: unable to upsert provides_ecs, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(providePrimaryKeyColumns))
			copy(conflict, providePrimaryKeyColumns)
		}
		cache.query = buildUpsertQueryPostgres(dialect, "\"provides_ecs\"", updateOnConflict, ret, update, conflict, insert)

		cache.valueMapping, err = queries.BindMapping(provideType, provideMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(provideType, provideMapping, ret)
			if err != nil {
				return err
			}
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)
	var returns []interface{}
	if len(cache.retMapping) != 0 {
		returns = queries.PtrsFromMapping(value, cache.retMapping)
	}

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, vals)
	}
	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(returns...)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil // Postgres doesn't return anything when there's no update
		}
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}
	if err != nil {
		return errors.Wrap(err, "models: unable to upsert provides_ecs")
	}

	if !cached {
		provideUpsertCacheMut.Lock()
		provideUpsertCache[key] = cache
		provideUpsertCacheMut.Unlock()
	}

	return o.doAfterUpsertHooks(ctx, exec)
}

// Delete deletes a single Provide record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *Provide) Delete(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if o == nil {
		return 0, errors.New("models: no Provide provided for delete")
	}

	if err := o.doBeforeDeleteHooks(ctx, exec); err != nil {
		return 0, err
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), providePrimaryKeyMapping)
	sql := "DELETE FROM \"provides_ecs\" WHERE \"id\"=$1"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete from provides_ecs")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by delete for provides_ecs")
	}

	if err := o.doAfterDeleteHooks(ctx, exec); err != nil {
		return 0, err
	}

	return rowsAff, nil
}

// DeleteAll deletes all matching rows.
func (q provideQuery) DeleteAll(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if q.Query == nil {
		return 0, errors.New("models: no provideQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	result, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete all from provides_ecs")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by deleteall for provides_ecs")
	}

	return rowsAff, nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o ProvideSlice) DeleteAll(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if len(o) == 0 {
		return 0, nil
	}

	if len(provideBeforeDeleteHooks) != 0 {
		for _, obj := range o {
			if err := obj.doBeforeDeleteHooks(ctx, exec); err != nil {
				return 0, err
			}
		}
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), providePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"provides_ecs\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, providePrimaryKeyColumns, len(o))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete all from provide slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by deleteall for provides_ecs")
	}

	if len(provideAfterDeleteHooks) != 0 {
		for _, obj := range o {
			if err := obj.doAfterDeleteHooks(ctx, exec); err != nil {
				return 0, err
			}
		}
	}

	return rowsAff, nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *Provide) Reload(ctx context.Context, exec boil.ContextExecutor) error {
	ret, err := FindProvide(ctx, exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *ProvideSlice) ReloadAll(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := ProvideSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), providePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"provides_ecs\".* FROM \"provides_ecs\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, providePrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(ctx, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in ProvideSlice")
	}

	*o = slice

	return nil
}

// ProvideExists checks if the Provide row exists.
func ProvideExists(ctx context.Context, exec boil.ContextExecutor, iD int) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"provides_ecs\" where \"id\"=$1 limit 1)"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, iD)
	}
	row := exec.QueryRowContext(ctx, sql, iD)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if provides_ecs exists")
	}

	return exists, nil
}

// Exists checks if the Provide row exists.
func (o *Provide) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	return ProvideExists(ctx, exec, o.ID)
}
