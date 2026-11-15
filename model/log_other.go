package model

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
)

const (
	logOtherAdminInfoKey    = "admin_info"
	logOtherRootInfoKey     = "root_info"
	logOtherAuditInfoKey    = "audit_info"
	logOtherContentPartsKey = "content_parts"
	// logOtherExprInternalKey marks, beside expr_b64, a consume row whose
	// model had a public price expression when the row was written.
	logOtherExprInternalKey = "expr_internal"
)

// legacySensitiveLogOtherKeys are historical top-level fields that must never
// be written via SetPublic and must be stripped from user-visible projections.
var legacySensitiveLogOtherKeys = []string{
	"channel_id",
	"channel_name",
	"channel_type",
	"reject_reason",
}

type logOtherVisibility int

const (
	logOtherVisibilityUser logOtherVisibility = iota
	logOtherVisibilityAdmin
	logOtherVisibilityRoot
)

// LogOther separates usage-log metadata by the audience allowed to see it.
// Its maps stay private so callers cannot accidentally place privileged fields
// in the user-visible top level.
type LogOther struct {
	public    map[string]any
	adminInfo map[string]any
	rootInfo  map[string]any
	auditInfo map[string]any
}

func NewLogOther() *LogOther {
	return &LogOther{}
}

func isReservedLogOtherKey(key string) bool {
	switch key {
	case logOtherAdminInfoKey, logOtherRootInfoKey, logOtherAuditInfoKey:
		return true
	default:
		return slices.Contains(legacySensitiveLogOtherKeys, key)
	}
}

// SetPublic records metadata that log owners may receive from self/token log APIs.
// It rejects role-scoped and legacy-sensitive keys so new writers cannot recreate
// the historical channel/reject-reason leak.
func (o *LogOther) SetPublic(key string, value any) bool {
	if o == nil || key == "" || isReservedLogOtherKey(key) {
		return false
	}
	if o.public == nil {
		o.public = make(map[string]any)
	}
	o.public[key] = value
	return true
}

func (o *LogOther) MergePublic(values map[string]any) {
	for key, value := range values {
		o.SetPublic(key, value)
	}
}

// setContent stores structured content under content_parts, which the web
// console renders in the viewer's language, and returns the English text kept
// in the content column for exports and API callers.
func (o *LogOther) setContent(parts []*common.Message) string {
	if len(parts) == 0 {
		return ""
	}
	o.SetPublic(logOtherContentPartsKey, parts)
	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		texts = append(texts, part.Error())
	}
	return strings.Join(texts, ", ")
}

// MarkBillingExprInternal records, beside the billing expression, that the
// row's billing expression is internal, so the row keeps it from everyone who
// is not an administrator even after the model's public expression is removed.
// A task row passes the model it maps to; any other row passes "".
func (o *LogOther) MarkBillingExprInternal(modelName, mappedModel string) {
	if billingExprInternal(modelName, mappedModel) {
		o.SetPublic(logOtherExprInternalKey, true)
	}
}

// billingExprInternal reports whether the billing expression of a consume row
// of modelName is internal: the model has a public price expression, shown in
// place of the billing expression to everyone who is not an administrator. A
// task of a model without expression billing of its own is priced by the model
// it maps to, so mappedModel's public expression counts then.
func billingExprInternal(modelName, mappedModel string) bool {
	if _, public := billing_setting.GetPublicExpr(modelName); public {
		return true
	}
	if mappedModel == "" || mappedModel == modelName || billing_setting.GetBillingMode(modelName) == billing_setting.BillingModeTieredExpr {
		return false
	}
	_, public := billing_setting.GetPublicExpr(mappedModel)
	return public
}

// internalBillingExprKeys are the fields of a consume row that come from its
// billing expression rather than from the charge: the expression, the request
// rules it traced and the label of the tier that matched, which is chosen with
// the expression and can name an internal condition.
var internalBillingExprKeys = []string{"expr_b64", "request_rules", "matched_tier"}

// WithholdInternalBillingExprs removes the internalBillingExprKeys from every
// row whose expression is internal: the row was marked when it was written, or
// its model has a public price expression now. Log controllers apply it to the
// rows they return to a caller who is not an administrator. The charge and
// every other field stay, and rows that are not internal are left byte for
// byte.
func WithholdInternalBillingExprs(logs []*Log) {
	for _, log := range logs {
		if !slices.ContainsFunc(internalBillingExprKeys, func(key string) bool { return strings.Contains(log.Other, `"`+key+`"`) }) {
			continue
		}
		var values map[string]json.RawMessage
		if err := common.UnmarshalJsonStr(log.Other, &values); err != nil {
			log.Other = "{}"
			continue
		}
		var marked, task bool
		var mappedModel string
		_ = common.Unmarshal(values[logOtherExprInternalKey], &marked)
		if _ = common.Unmarshal(values["is_task"], &task); task {
			_ = common.Unmarshal(values["upstream_model_name"], &mappedModel)
		}
		withheld := slices.ContainsFunc(internalBillingExprKeys, func(key string) bool {
			_, exists := values[key]
			return exists
		})
		if !withheld || !marked && !billingExprInternal(log.ModelName, mappedModel) {
			continue
		}
		for _, key := range internalBillingExprKeys {
			delete(values, key)
		}
		encoded, err := common.Marshal(values)
		if err != nil {
			log.Other = "{}"
			continue
		}
		log.Other = string(encoded)
	}
}

func (o *LogOther) SetAdmin(key string, value any) bool {
	if o == nil || key == "" {
		return false
	}
	if o.adminInfo == nil {
		o.adminInfo = make(map[string]any)
	}
	o.adminInfo[key] = value
	return true
}

func (o *LogOther) MergeAdmin(values map[string]any) {
	for key, value := range values {
		o.SetAdmin(key, value)
	}
}

func (o *LogOther) SetRoot(key string, value any) bool {
	if o == nil || key == "" {
		return false
	}
	if o.rootInfo == nil {
		o.rootInfo = make(map[string]any)
	}
	o.rootInfo[key] = value
	return true
}

func (o *LogOther) MergeRoot(values map[string]any) {
	for key, value := range values {
		o.SetRoot(key, value)
	}
}

func (o *LogOther) SetAudit(key string, value any) bool {
	if o == nil || key == "" {
		return false
	}
	if o.auditInfo == nil {
		o.auditInfo = make(map[string]any)
	}
	o.auditInfo[key] = value
	return true
}

func (o *LogOther) MergeAudit(values map[string]any) {
	for key, value := range values {
		o.SetAudit(key, value)
	}
}

func copyLogOtherMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	copyValues := make(map[string]any, len(values))
	maps.Copy(copyValues, values)
	return copyValues
}

func (o *LogOther) toMap() map[string]any {
	result := make(map[string]any)
	if o == nil {
		return result
	}

	maps.Copy(result, o.public)
	if adminInfo := copyLogOtherMap(o.adminInfo); len(adminInfo) > 0 {
		result[logOtherAdminInfoKey] = adminInfo
	}
	if auditInfo := copyLogOtherMap(o.auditInfo); len(auditInfo) > 0 {
		result[logOtherAuditInfoKey] = auditInfo
	}
	if rootInfo := copyLogOtherMap(o.rootInfo); len(rootInfo) > 0 {
		result[logOtherRootInfoKey] = rootInfo
	}
	return result
}

func (o *LogOther) jsonString() string {
	if o == nil {
		return ""
	}
	data, err := common.Marshal(o.toMap())
	if err != nil {
		common.SysError(common.LogText("failed to marshal log other: %s", err.Error()))
		return ""
	}
	return string(data)
}

// JSONString returns the complete stored representation, including all
// privileged scopes. API projections must use the role-specific formatter.
func (o *LogOther) JSONString() string {
	return o.jsonString()
}

// Snapshot returns a detached top-level view for tests and read-only
// inspection. Mutating it cannot add or replace fields in LogOther.
func (o *LogOther) Snapshot() map[string]any {
	if o == nil {
		return nil
	}
	return o.toMap()
}

func (o *LogOther) MarshalJSON() ([]byte, error) {
	return common.Marshal(o.toMap())
}

func normalizeLegacyRejectReason(values map[string]json.RawMessage) bool {
	rejectReason, ok := values["reject_reason"]
	if !ok {
		return false
	}

	adminInfo := make(map[string]json.RawMessage)
	if rawAdminInfo, exists := values[logOtherAdminInfoKey]; exists {
		_ = common.Unmarshal(rawAdminInfo, &adminInfo)
	}
	if adminInfo == nil {
		adminInfo = make(map[string]json.RawMessage)
	}
	if _, exists := adminInfo["reject_reason"]; !exists {
		adminInfo["reject_reason"] = rejectReason
	}
	encodedAdminInfo, err := common.Marshal(adminInfo)
	if err != nil {
		return false
	}
	values[logOtherAdminInfoKey] = encodedAdminInfo
	delete(values, "reject_reason")
	return true
}

// formatLogOtherJSON applies the role projection while keeping untouched JSON
// values as RawMessage. This preserves integers larger than JavaScript's safe
// range instead of round-tripping them through float64.
func formatLogOtherJSON(value string, visibility logOtherVisibility) string {
	if value == "" {
		return ""
	}

	var values map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(value, &values); err != nil {
		if visibility == logOtherVisibilityRoot {
			return value
		}
		return "{}"
	}

	changed := false
	if visibility == logOtherVisibilityUser {
		for _, key := range []string{logOtherAdminInfoKey, logOtherRootInfoKey, logOtherAuditInfoKey} {
			if _, exists := values[key]; exists {
				delete(values, key)
				changed = true
			}
		}
		for _, key := range legacySensitiveLogOtherKeys {
			if _, exists := values[key]; exists {
				delete(values, key)
				changed = true
			}
		}
	} else {
		changed = normalizeLegacyRejectReason(values)
		if visibility == logOtherVisibilityAdmin {
			if _, exists := values[logOtherRootInfoKey]; exists {
				delete(values, logOtherRootInfoKey)
				changed = true
			}
		}
	}

	if !changed {
		return value
	}
	formatted, err := common.Marshal(values)
	if err != nil {
		if visibility == logOtherVisibilityRoot {
			return value
		}
		return "{}"
	}
	return string(formatted)
}
