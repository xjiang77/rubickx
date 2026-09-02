package server

import "context"

const (
	AlgorithmFixedWindow        = "fixed-window"
	AlgorithmSlidingWindowLog   = "sliding-window-log"
	AlgorithmSlidingWindowCount = "sliding-window-counter"
	AlgorithmTokenBucket        = "token-bucket"
	AlgorithmLeakyBucket        = "leaky-bucket"
	LanguageGo                  = "go"
	LanguagePython              = "python"
	LanguageJava                = "java"
	LanguageJavaScript          = "javascript"
)

type RunRequest struct {
	ScenarioID      string             `json:"scenarioId"`
	Algorithm       string             `json:"algorithm"`
	Language        string             `json:"language"`
	Config          map[string]float64 `json:"config"`
	RequestTimeline []RequestPoint     `json:"requestTimeline"`
	StoreMode       string             `json:"storeMode,omitempty"`
}

type RequestPoint struct {
	AtMs int64   `json:"atMs"`
	Cost float64 `json:"cost"`
	Key  string  `json:"key"`
}

type Decision struct {
	Allowed      bool    `json:"allowed"`
	Remaining    float64 `json:"remaining"`
	RetryAfterMs float64 `json:"retryAfterMs"`
	ResetAtMs    float64 `json:"resetAtMs"`
	Reason       string  `json:"reason"`
}

type SourceAnchor struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

type TraceEvent struct {
	Seq         int            `json:"seq"`
	StepID      string         `json:"stepId"`
	Actor       string         `json:"actor"`
	TimestampMs int64          `json:"timestampMs"`
	Before      map[string]any `json:"before"`
	After       map[string]any `json:"after"`
	Decision    *Decision      `json:"decision,omitempty"`
	Reason      string         `json:"reason"`
	Source      SourceAnchor   `json:"source"`
}

type SourceDocument struct {
	Language string         `json:"language"`
	Path     string         `json:"path"`
	Content  string         `json:"content"`
	Anchors  map[string]int `json:"anchors"`
}

type RunResponse struct {
	RunID     string         `json:"runId,omitempty"`
	Language  string         `json:"language,omitempty"`
	Algorithm string         `json:"algorithm,omitempty"`
	Events    []TraceEvent   `json:"events"`
	Decisions []Decision     `json:"decisions"`
	Source    SourceDocument `json:"source"`
}

type LanguageRunner interface {
	Run(context.Context, RunRequest) (RunResponse, error)
}

type ScenarioBrief struct {
	Policy        string              `json:"policy,omitempty"`
	Traffic       string              `json:"traffic"`
	Expected      ScenarioExpectation `json:"expected"`
	ReplicaScoped bool                `json:"replicaScoped,omitempty"`
	Conceptual    bool                `json:"conceptual,omitempty"`
}

type ScenarioExpectation struct {
	Summary    string                 `json:"summary"`
	Admissions []string               `json:"admissions,omitempty"`
	Cases      []ScenarioExpectedCase `json:"cases,omitempty"`
}

type ScenarioExpectedCase struct {
	When   string `json:"when"`
	Result string `json:"result"`
	Kind   string `json:"kind"`
}

type CatalogItem struct {
	ID              string             `json:"id"`
	Label           string             `json:"label"`
	Description     string             `json:"description,omitempty"`
	Tier            string             `json:"tier,omitempty"`
	Algorithm       string             `json:"algorithm,omitempty"`
	DefaultLanguage string             `json:"defaultLanguage,omitempty"`
	DefaultConfig   map[string]float64 `json:"defaultConfig,omitempty"`
	RequestTimeline []RequestPoint     `json:"requestTimeline,omitempty"`
	Lesson          string             `json:"lesson,omitempty"`
	Brief           *ScenarioBrief     `json:"brief,omitempty"`
	Debuggable      bool               `json:"debuggable,omitempty"`
}

type Catalog struct {
	Algorithms []CatalogItem `json:"algorithms"`
	Languages  []CatalogItem `json:"languages"`
	Scenarios  []CatalogItem `json:"scenarios"`
	Modes      []string      `json:"modes"`
}
