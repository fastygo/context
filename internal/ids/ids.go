// Package ids defines opaque typed identifiers for the context core.
package ids

import "fmt"

// Typed identifiers keep domain language explicit without premature UUID helpers.
type (
	TenantID     string
	ProjectID    string
	SourceID     string
	ArtifactID   string
	ChunkID      string
	SnapshotID   string
	RunID        string
	ToolCallID   string
	PackID       string
	TaskID       string
	EvalID       string
	TraceEventID string
	TokenID      string
	SenseID      string
	ConceptID    string
	AttestationID string
	VariantID    string
	MWEID        string
	LexiconSourceID string
	PolicyID     string
	FocusID      string
	PlanID       string
	ModelCallID  string
	GraphNodeID  string
	GraphEdgeID  string
	ContextRefID string
	PathAliasID  string
	QueryID      string
	ExpansionID  string
	MorphAnalysisID string
	LexemeID        string
	JobID           string
	ScheduleID      string
)

// Validate reports whether the identifier is non-empty.
func Validate(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s: empty id", name)
	}
	return nil
}

func (id TenantID) Validate() error      { return Validate("tenant_id", string(id)) }
func (id ProjectID) Validate() error     { return Validate("project_id", string(id)) }
func (id SourceID) Validate() error      { return Validate("source_id", string(id)) }
func (id ArtifactID) Validate() error    { return Validate("artifact_id", string(id)) }
func (id ChunkID) Validate() error       { return Validate("chunk_id", string(id)) }
func (id SnapshotID) Validate() error    { return Validate("snapshot_id", string(id)) }
func (id RunID) Validate() error         { return Validate("run_id", string(id)) }
func (id ToolCallID) Validate() error    { return Validate("tool_call_id", string(id)) }
func (id PackID) Validate() error        { return Validate("pack_id", string(id)) }
func (id TaskID) Validate() error        { return Validate("task_id", string(id)) }
func (id EvalID) Validate() error        { return Validate("eval_id", string(id)) }
func (id TraceEventID) Validate() error  { return Validate("trace_event_id", string(id)) }
func (id TokenID) Validate() error       { return Validate("token_id", string(id)) }
func (id SenseID) Validate() error       { return Validate("sense_id", string(id)) }
func (id ConceptID) Validate() error     { return Validate("concept_id", string(id)) }
func (id AttestationID) Validate() error { return Validate("attestation_id", string(id)) }
func (id VariantID) Validate() error     { return Validate("variant_id", string(id)) }
func (id MWEID) Validate() error         { return Validate("mwe_id", string(id)) }
func (id LexiconSourceID) Validate() error {
	return Validate("lexicon_source_id", string(id))
}
func (id PolicyID) Validate() error        { return Validate("policy_id", string(id)) }
func (id FocusID) Validate() error         { return Validate("focus_id", string(id)) }
func (id PlanID) Validate() error          { return Validate("plan_id", string(id)) }
func (id ModelCallID) Validate() error     { return Validate("model_call_id", string(id)) }
func (id GraphNodeID) Validate() error     { return Validate("graph_node_id", string(id)) }
func (id GraphEdgeID) Validate() error     { return Validate("graph_edge_id", string(id)) }
func (id ContextRefID) Validate() error    { return Validate("context_ref_id", string(id)) }
func (id PathAliasID) Validate() error     { return Validate("path_alias_id", string(id)) }
func (id QueryID) Validate() error         { return Validate("query_id", string(id)) }
func (id ExpansionID) Validate() error     { return Validate("expansion_id", string(id)) }
func (id MorphAnalysisID) Validate() error { return Validate("morph_analysis_id", string(id)) }
func (id LexemeID) Validate() error        { return Validate("lexeme_id", string(id)) }
func (id JobID) Validate() error           { return Validate("job_id", string(id)) }
func (id ScheduleID) Validate() error      { return Validate("schedule_id", string(id)) }
