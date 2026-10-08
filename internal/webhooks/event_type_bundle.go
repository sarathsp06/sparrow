package webhooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	jsonschema "github.com/kaptinlin/jsonschema"
	"github.com/sarathsp06/schemagen"

	"github.com/sarathsp06/sparrow"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
	"github.com/sarathsp06/sparrow/pkg/template"
)

// Event type bundle: a JSON document holding many event type definitions, so
// definitions can be exported from one environment and imported into
// another.
const (
	BundleAPIVersion = "sparrow/v1"
	BundleKind       = "EventTypeList"
	// BundleFormat is the bundle format this server writes and the newest it
	// reads.
	BundleFormat = 1
	// MaxBundleItems caps how many event types one import may contain, which
	// bounds the import transaction.
	MaxBundleItems = 500
)

// Warning codes for an import's version stamp. Each is acknowledged by name.
const (
	StampVersionDiffers    = "version_differs"
	StampFormatUnsupported = "format_unsupported"
	StampItemsChanged      = "items_changed"
	// BlockedByBreaking is reported in ImportResult.BlockedBy when an item's
	// schema change is breaking and AllowBreaking was not set.
	BlockedByBreaking = "breaking"
)

// BundleStamp records which Sparrow produced a bundle. It is a compatibility
// hint, not a signature: anyone can edit a file and recompute it.
type BundleStamp struct {
	SparrowVersion string
	Format         int
	// SHA256 is the hex digest of the canonical encoding of the items.
	SHA256 string
}

// EventTypeExportSelection says which event types to export. Exactly one of
// Names, Prefix or All is used.
type EventTypeExportSelection struct {
	Names  []string
	Prefix string
	All    bool
}

// EventTypeImportOptions control an import.
type EventTypeImportOptions struct {
	// DryRun computes the full result and writes nothing.
	DryRun bool
	// Acknowledge lists the stamp warning codes the caller accepts.
	Acknowledge []string
	// AllowBreaking applies breaking schema changes to subscribed types.
	AllowBreaking bool
	// PauseFailing pauses, in the same transaction, every subscription whose
	// template fails the pre-check against the new version. Off by default:
	// a mismatched template then fails its deliveries visibly instead.
	PauseFailing bool
}

// StampCheck is the outcome of comparing an import's stamp with this server.
type StampCheck struct {
	// Status is "valid", "unsigned" (no stamp) or "warning".
	Status     string
	ExportedBy string
	Server     string
	Warnings   []string
}

// TemplateCheckFailure is a subscription whose transform template no longer
// renders against the new version of the event type.
type TemplateCheckFailure struct {
	SubscriptionID string
	Consumer       string
	WebhookID      string
	// Payload is "sample" (every field) or "required_only".
	Payload string
	Error   string
}

// TemplateCheck summarizes how an event type's subscriptions fare against a
// new version.
type TemplateCheck struct {
	Failures         []TemplateCheckFailure
	Passed           int
	WithoutTransform int
	CatchAll         int
	// Paused lists subscriptions this import paused (PauseFailing).
	Paused []string
}

// EventTypeImportItem is the result for one bundle entry.
type EventTypeImportItem struct {
	*EventTypeSaveResult
	// Templates is set for new versions.
	Templates *TemplateCheck
}

// EventTypeImportResult describes an import, applied or not.
type EventTypeImportResult struct {
	Applied    bool
	DryRun     bool
	ImportedAt time.Time
	Stamp      StampCheck
	Items      []EventTypeImportItem
	// BlockedBy lists why nothing was written: unacknowledged stamp warning
	// codes, and BlockedByBreaking.
	BlockedBy []string
}

// bundleItem is the canonical encoding of one definition, used for the stamp
// digest. Field order is fixed and Go sorts map keys, so equal definitions
// always encode to the same bytes.
type bundleItem struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Schema      map[string]any    `json:"event_schema,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Active      bool              `json:"active"`
}

// BundleDigest returns the stamp digest of a list of definitions: sha256 of
// their canonical JSON encoding. Formatting of the source file does not
// affect it.
func BundleDigest(items []EventTypeDefinition) string {
	canon := make([]bundleItem, len(items))
	for i, it := range items {
		canon[i] = bundleItem(it)
		if len(canon[i].Schema) == 0 {
			canon[i].Schema = nil
		}
		if len(canon[i].Metadata) == 0 {
			canon[i].Metadata = nil
		}
	}
	b, _ := json.Marshal(canon) // only maps, strings and bools: cannot fail
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ExportEventTypes returns the current definitions of the selected event
// types, sorted by name, and the stamp for the bundle. System event types
// (sparrow.*) are never exported. An unknown name fails the whole export.
func (s *WebhookService) ExportEventTypes(ctx context.Context, sel EventTypeExportSelection) ([]EventTypeDefinition, BundleStamp, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ExportEventTypes")
	defer span.End()

	selectors := 0
	if len(sel.Names) > 0 {
		selectors++
	}
	if sel.Prefix != "" {
		selectors++
	}
	if sel.All {
		selectors++
	}
	if selectors != 1 {
		return nil, BundleStamp{}, svcerrors.Error(svcerrors.InvalidArgument, "choose exactly one of names, prefix or all")
	}
	tenantID := tenant.DefaultTenantID

	var regs []*store.EventRegistration
	if len(sel.Names) > 0 {
		for _, name := range sel.Names {
			if IsReservedEventName(name) {
				return nil, BundleStamp{}, reservedEventNameError(name)
			}
			reg, err := s.webhookRepo.GetEventByName(ctx, tenantID, name)
			if err != nil {
				return nil, BundleStamp{}, fmt.Errorf("failed to load event type: %w", err)
			}
			if reg == nil {
				return nil, BundleStamp{}, svcerrors.Errorf(svcerrors.NotFound, "event type %q not found", name)
			}
			regs = append(regs, reg)
		}
	} else {
		for offset := 0; ; offset += maxPageLimit {
			page, _, err := s.webhookRepo.ListEventsPaginated(ctx, tenantID, store.EventTypeFilter{System: new(bool)}, maxPageLimit, offset)
			if err != nil {
				return nil, BundleStamp{}, fmt.Errorf("failed to list event types: %w", err)
			}
			for _, reg := range page {
				if !strings.HasPrefix(reg.Name, sel.Prefix) {
					continue
				}
				regs = append(regs, reg)
			}
			if len(page) < maxPageLimit {
				break
			}
		}
	}

	defs := make([]EventTypeDefinition, 0, len(regs))
	seen := map[string]bool{}
	for _, reg := range regs {
		if seen[reg.Name] {
			continue
		}
		seen[reg.Name] = true
		defs = append(defs, EventTypeDefinition{
			Name:        reg.Name,
			Description: reg.Description,
			Schema:      reg.Schema,
			Metadata:    reg.Metadata,
			Active:      reg.Active,
		})
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })

	return defs, BundleStamp{SparrowVersion: sparrow.Version, Format: BundleFormat, SHA256: BundleDigest(defs)}, nil
}

// checkStamp compares an import's stamp with this server.
func checkStamp(stamp *BundleStamp, items []EventTypeDefinition) StampCheck {
	check := StampCheck{Server: sparrow.Version}
	if stamp == nil {
		check.Status = "unsigned"
		return check
	}
	check.ExportedBy = stamp.SparrowVersion
	if stamp.SparrowVersion != sparrow.Version {
		check.Warnings = append(check.Warnings, StampVersionDiffers)
	}
	if stamp.Format < 1 || stamp.Format > BundleFormat {
		check.Warnings = append(check.Warnings, StampFormatUnsupported)
	}
	if stamp.SHA256 != BundleDigest(items) {
		check.Warnings = append(check.Warnings, StampItemsChanged)
	}
	check.Status = "valid"
	if len(check.Warnings) > 0 {
		check.Status = "warning"
	}
	return check
}

// validateBundle rejects a bundle that cannot be imported at all. Every
// problem is reported, so a file can be fixed in one pass.
func validateBundle(items []EventTypeDefinition) error {
	if len(items) == 0 {
		return svcerrors.Error(svcerrors.InvalidArgument, "bundle has no items")
	}
	if len(items) > MaxBundleItems {
		return svcerrors.Errorf(svcerrors.InvalidArgument, "bundle has %d items; at most %d are allowed per import", len(items), MaxBundleItems)
	}
	var problems []string
	seen := map[string]bool{}
	for _, it := range items {
		if err := validateEventTypeName(it.Name); err != nil {
			problems = append(problems, fmt.Sprintf("%q: %s", it.Name, svcerrors.Classify(err, "").ClientMessage()))
			continue
		}
		if seen[it.Name] {
			problems = append(problems, fmt.Sprintf("%q: appears more than once", it.Name))
		}
		seen[it.Name] = true
		if len(it.Schema) > 0 {
			if err := compileSchema(it.Schema); err != nil {
				problems = append(problems, fmt.Sprintf("%q: event_schema does not compile: %v", it.Name, err))
			}
		}
	}
	if len(problems) > 0 {
		return svcerrors.Errorf(svcerrors.InvalidArgument, "bundle is invalid, nothing was imported: %s", strings.Join(problems, "; "))
	}
	return nil
}

func compileSchema(schema map[string]any) error {
	b, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	_, err = jsonschema.NewCompiler().Compile(b)
	return err
}

// errImportRollback aborts the import transaction after the result has been
// computed (dry run, or blocked).
var errImportRollback = errors.New("import rolled back")

// ImportEventTypes applies a bundle in one transaction: every item is saved
// with the usual rules (see planEventTypeSave), or nothing is. Types absent
// from the bundle are never touched: an import does not delete.
//
// The result is computed in full either way. Nothing is written when DryRun
// is set, when a stamp warning is not in Acknowledge, or when an item's
// schema change is breaking for its subscriptions and AllowBreaking is not
// set; BlockedBy then says why.
func (s *WebhookService) ImportEventTypes(ctx context.Context, items []EventTypeDefinition, stamp *BundleStamp, opts EventTypeImportOptions) (*EventTypeImportResult, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ImportEventTypes")
	defer span.End()

	if err := validateBundle(items); err != nil {
		return nil, err
	}
	sorted := slices.Clone(items)
	// Lock rows in name order so overlapping imports cannot deadlock.
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	tenantID := tenant.DefaultTenantID
	result := &EventTypeImportResult{DryRun: opts.DryRun, Stamp: checkStamp(stamp, items)}
	for _, w := range result.Stamp.Warnings {
		if !slices.Contains(opts.Acknowledge, w) {
			result.BlockedBy = append(result.BlockedBy, w)
		}
	}

	run := func() error {
		result.Items = result.Items[:0]
		result.BlockedBy = slices.DeleteFunc(result.BlockedBy, func(b string) bool { return b == BlockedByBreaking })
		return s.webhookRepo.RunInTransaction(func(tx store.RepositoryInterface) error {
			breaking := false
			for _, def := range sorted {
				res, err := s.saveEventTypeTx(ctx, tx, tenantID, def, saveUpsert, saveEventTypeOptions{AllowBreaking: opts.AllowBreaking})
				if err != nil {
					return fmt.Errorf("%s: %w", def.Name, err)
				}
				item := EventTypeImportItem{EventTypeSaveResult: res}
				if res.Action == EventTypeNewVersion {
					item.Templates = checkTemplates(def, res.AffectedSubscriptions)
					if opts.PauseFailing && !res.Blocked {
						if err := pauseFailing(ctx, tx, tenantID, res, item.Templates); err != nil {
							return err
						}
					}
				}
				breaking = breaking || res.Blocked
				result.Items = append(result.Items, item)
			}
			if breaking {
				result.BlockedBy = append(result.BlockedBy, BlockedByBreaking)
			}
			if opts.DryRun || len(result.BlockedBy) > 0 {
				return errImportRollback
			}
			return nil
		})
	}

	err := run()
	if errors.Is(err, storage.ErrAlreadyExists) {
		// A concurrent writer created one of the types first; the retry sees it.
		err = run()
	}
	switch {
	case errors.Is(err, errImportRollback):
		return result, nil
	case err != nil:
		return nil, err
	}
	result.Applied = true
	result.ImportedAt = time.Now().UTC()
	s.logger.InfoContext(ctx, "Imported event types", "count", len(result.Items))
	return result, nil
}

// pauseFailing pauses each subscription that failed the template check,
// recording the import as the reason.
func pauseFailing(ctx context.Context, tx store.RepositoryInterface, tenantID uuid.UUID, res *EventTypeSaveResult, check *TemplateCheck) error {
	reason := fmt.Sprintf("event type %s moved to v%d by import; its template failed the pre-check", res.Name, res.Version)
	seen := map[string]bool{}
	for _, f := range check.Failures {
		if seen[f.SubscriptionID] {
			continue
		}
		seen[f.SubscriptionID] = true
		id, err := uuid.Parse(f.SubscriptionID)
		if err != nil {
			return err
		}
		if _, err := tx.SetSubscriptionPaused(ctx, tenantID, id, true, reason); err != nil {
			return fmt.Errorf("pause subscription %s: %w", id, err)
		}
		check.Paused = append(check.Paused, f.SubscriptionID)
	}
	return nil
}

// checkTemplates renders every affected subscription's transform against the
// new version, strictly, with a full sample payload and with only the
// required fields. The second catches templates that read an optional field
// without guarding for its absence.
func checkTemplates(def EventTypeDefinition, subs []*store.EventSubscription) *TemplateCheck {
	check := &TemplateCheck{}
	if len(subs) == 0 {
		return check
	}
	payloads := []struct {
		name string
		data map[string]any
	}{
		{"sample", generatePayload(def.Schema, true)},
		{"required_only", generatePayload(def.Schema, false)},
	}
	engine := template.NewTemplateEngine()
	strict := template.ExecOptions{StrictMissingKeys: true}
	for _, sub := range subs {
		if sub.EventName == store.CatchAllEventName {
			check.CatchAll++
		}
		if !sub.TransformEnabled || sub.TransformTemplate == "" {
			check.WithoutTransform++
			continue
		}
		failed := false
		for _, p := range payloads {
			ctx := template.NewWebhookTemplateContext("import-check", def.Name, time.Now().UTC().Format(time.RFC3339), 1, p.data)
			if _, err := engine.TransformPayloadWith(sub.TransformTemplate, ctx, strict); err != nil {
				check.Failures = append(check.Failures, TemplateCheckFailure{
					SubscriptionID: sub.ID.String(),
					Consumer:       sub.Consumer,
					WebhookID:      sub.WebhookID.String(),
					Payload:        p.name,
					Error:          err.Error(),
				})
				failed = true
				break
			}
		}
		if !failed {
			check.Passed++
		}
	}
	return check
}

// generatePayload builds an example payload from a schema: every field when
// all is true, only required fields otherwise. Generation errors yield an
// empty payload, which the strict render then exercises.
func generatePayload(schema map[string]any, all bool) map[string]any {
	if len(schema) == 0 {
		return map[string]any{}
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return map[string]any{}
	}
	sample, err := schemagen.NewGenerator().SetSeed(1).SetGenerateAllFields(all).Generate(b)
	if err != nil {
		return map[string]any{}
	}
	if m, ok := sample.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
