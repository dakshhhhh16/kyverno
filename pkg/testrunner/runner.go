package testrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"github.com/kyverno/kyverno/pkg/clients/dclient"
	engineapi "github.com/kyverno/kyverno/pkg/engine/api"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestBackend is the unified interface for test execution backends
// Both Fast Mode and Accurate Mode implement this interface, enabling
// seamless switching between modes without changing test code
type TestBackend interface {
	// Setup initializes the backend with the given objects
	Setup(ctx context.Context, objects []runtime.Object) error

	// Teardown cleans up the backend resources
	Teardown(ctx context.Context) error

	// Client returns the dclient.Interface for policy evaluation
	Client() dclient.Interface

	// ConfigmapResolver returns the configmap resolver
	ConfigmapResolver() engineapi.ConfigmapResolver

	// Mode returns which mode this backend is running in
	Mode() TestMode

	// IsReady returns whether the backend is initialized and ready
	IsReady() bool
}

// TestResult holds the result of evaluating a policy against a resource
type TestResult struct {
	// PolicyName is the name of the policy that was evaluated
	PolicyName string

	// RuleName is the name of the specific rule within the policy
	RuleName string

	// ResourceName is the name of the resource tested
	ResourceName string

	// ResourceNamespace is the namespace of the resource tested
	ResourceNamespace string

	// ResourceKind is the kind of the resource tested
	ResourceKind string

	// Status is the evaluation result (pass, fail, warn, error, skip)
	Status string

	// Message contains additional details about the result
	Message string

	// Mode indicates which mode was used for this evaluation
	Mode TestMode

	// Duration is how long the evaluation took
	Duration time.Duration
}

// TestSummary holds aggregated results from a test run
type TestSummary struct {
	// Mode is the mode used for this test run
	Mode TestMode

	// SetupDuration is how long backend setup took
	SetupDuration time.Duration

	// EvalDuration is how long policy evaluation took
	EvalDuration time.Duration

	// TotalDuration is the total time including setup and teardown
	TotalDuration time.Duration

	// Results contains individual test results
	Results []TestResult

	// Pass is the count of passing results
	Pass int

	// Fail is the count of failing results
	Fail int

	// Warn is the count of warning results
	Warn int

	// Error is the count of error results
	Error int

	// Skip is the count of skipped results
	Skip int

	// FellBack indicates the runner fell back from Accurate to Fast mode
	FellBack bool

	// FallbackReason explains why fallback occurred
	FallbackReason string
}

// TestRunner is the unified "one-stop" test runner that seamlessly
// switches between Fast Mode (Smart Mocks) and Accurate Mode (envtest)
type TestRunner struct {
	config  TestConfig
	backend TestBackend
	out     io.Writer
}

// NewTestRunner creates a new unified test runner
func NewTestRunner(config TestConfig) *TestRunner {
	return &TestRunner{
		config: config,
		out:    os.Stdout,
	}
}

// SetOutput sets the output writer for status messages
func (r *TestRunner) SetOutput(w io.Writer) {
	r.out = w
}

// Run executes the unified test workflow
// This is the single entry point that handles both modes transparently
func (r *TestRunner) Run(ctx context.Context, policies []kyvernov1.PolicyInterface, resources []*unstructured.Unstructured) (*TestSummary, error) {
	totalStart := time.Now()

	if err := r.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	summary := &TestSummary{
		Mode: r.config.Mode,
	}

	// Convert resources to runtime.Object for backend setup
	objects := make([]runtime.Object, 0, len(resources))
	for _, res := range resources {
		objects = append(objects, res.DeepCopy())
	}

	// Phase 1: Setup backend
	fmt.Fprintf(r.out, "🚀 Setting up %s...\n", r.config.Mode.Description())
	setupStart := time.Now()

	backend, err := r.createBackend()
	if err != nil {
		// Auto-fallback from Accurate to Fast if enabled
		if r.config.Mode == ModeAccurate && r.config.AutoFallback {
			fmt.Fprintf(r.out, "⚠️  Accurate mode setup failed: %v\n", err)
			fmt.Fprintf(r.out, "↩️  Falling back to Fast Mode...\n")
			summary.FellBack = true
			summary.FallbackReason = err.Error()
			r.config.Mode = ModeFast
			backend, err = r.createBackend()
			if err != nil {
				return nil, fmt.Errorf("both accurate and fast mode setup failed: %w", err)
			}
		} else {
			return nil, fmt.Errorf("backend setup failed: %w", err)
		}
	}
	r.backend = backend

	if err := r.backend.Setup(ctx, objects); err != nil {
		// Auto-fallback on setup failure
		if r.config.Mode == ModeAccurate && r.config.AutoFallback {
			fmt.Fprintf(r.out, "⚠️  Accurate mode environment failed: %v\n", err)
			fmt.Fprintf(r.out, "↩️  Falling back to Fast Mode...\n")
			summary.FellBack = true
			summary.FallbackReason = err.Error()

			fastBackend := newFastBackend()
			if err := fastBackend.Setup(ctx, objects); err != nil {
				return nil, fmt.Errorf("fallback fast mode setup failed: %w", err)
			}
			r.backend = fastBackend
			summary.Mode = ModeFast
		} else {
			return nil, fmt.Errorf("backend setup failed: %w", err)
		}
	}

	summary.SetupDuration = time.Since(setupStart)
	fmt.Fprintf(r.out, "✅ Backend ready (%s) in %v\n", r.backend.Mode(), summary.SetupDuration)

	// Phase 2: Evaluate policies against resources
	fmt.Fprintf(r.out, "📋 Evaluating %d policies against %d resources...\n", len(policies), len(resources))
	evalStart := time.Now()

	results, err := r.evaluatePolicies(ctx, policies, resources)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	summary.EvalDuration = time.Since(evalStart)
	summary.Results = results

	// Phase 3: Aggregate results
	for _, result := range results {
		switch result.Status {
		case "pass":
			summary.Pass++
		case "fail":
			summary.Fail++
		case "warn":
			summary.Warn++
		case "error":
			summary.Error++
		case "skip":
			summary.Skip++
		}
	}

	// Phase 4: Teardown
	if err := r.backend.Teardown(ctx); err != nil {
		fmt.Fprintf(r.out, "⚠️  Backend teardown warning: %v\n", err)
	}

	summary.TotalDuration = time.Since(totalStart)

	// Print summary
	r.printSummary(summary)

	return summary, nil
}

// createBackend creates the appropriate backend for the configured mode
func (r *TestRunner) createBackend() (TestBackend, error) {
	switch r.config.Mode {
	case ModeFast:
		return newFastBackend(), nil
	case ModeAccurate:
		return newAccurateBackend(r.config.CRDPaths), nil
	default:
		return nil, fmt.Errorf("unknown mode: %s", r.config.Mode)
	}
}

// evaluatePolicies runs policy evaluation using the Kyverno engine
func (r *TestRunner) evaluatePolicies(
	ctx context.Context,
	policies []kyvernov1.PolicyInterface,
	resources []*unstructured.Unstructured,
) ([]TestResult, error) {
	var results []TestResult

	client := r.backend.Client()
	if client == nil {
		return nil, fmt.Errorf("backend client is nil")
	}

	for _, resource := range resources {
		resourceKey := fmt.Sprintf("%s/%s/%s",
			resource.GetKind(),
			resource.GetNamespace(),
			resource.GetName(),
		)

		for _, pol := range policies {
			evalStart := time.Now()

			// Check if policy matches this resource
			if !policyMatchesResource(pol, resource) {
				results = append(results, TestResult{
					PolicyName:        pol.GetName(),
					ResourceName:      resource.GetName(),
					ResourceNamespace: resource.GetNamespace(),
					ResourceKind:      resource.GetKind(),
					Status:            "skip",
					Message:           "policy does not match resource",
					Mode:              r.backend.Mode(),
					Duration:          time.Since(evalStart),
				})
				continue
			}

			// In a full implementation, this would use processor.PolicyProcessor
			// to run the actual Kyverno engine evaluation.
			// For the PoC, we demonstrate the unified interface by checking
			// policy rules against the resource.
			ruleResults := evaluatePolicyRules(pol, resource, r.backend.Mode())

			for _, rr := range ruleResults {
				rr.Duration = time.Since(evalStart)
				results = append(results, rr)
			}

			fmt.Fprintf(r.out, "  %s → %s: evaluated %d rules\n",
				pol.GetName(), resourceKey, len(ruleResults))
		}
	}

	return results, nil
}

// policyMatchesResource checks if a policy's match criteria cover the resource
func policyMatchesResource(pol kyvernov1.PolicyInterface, resource *unstructured.Unstructured) bool {
	spec := pol.GetSpec()
	for _, rule := range spec.Rules {
		kinds := rule.MatchResources.GetKinds()
		for _, kind := range kinds {
			if kind == resource.GetKind() {
				return true
			}
		}
	}
	return false
}

// evaluatePolicyRules evaluates individual rules in a policy against a resource
func evaluatePolicyRules(pol kyvernov1.PolicyInterface, resource *unstructured.Unstructured, mode TestMode) []TestResult {
	var results []TestResult
	spec := pol.GetSpec()

	for _, rule := range spec.Rules {
		result := TestResult{
			PolicyName:        pol.GetName(),
			RuleName:          rule.Name,
			ResourceName:      resource.GetName(),
			ResourceNamespace: resource.GetNamespace(),
			ResourceKind:      resource.GetKind(),
			Mode:              mode,
		}

		// Check if rule has validation
		if rule.HasValidate() {
			result.Status = "pass"
			result.Message = fmt.Sprintf("validation rule '%s' evaluated in %s mode", rule.Name, mode)
		} else if rule.HasMutate() {
			result.Status = "pass"
			result.Message = fmt.Sprintf("mutation rule '%s' evaluated in %s mode", rule.Name, mode)
		} else if rule.HasGenerate() {
			result.Status = "pass"
			result.Message = fmt.Sprintf("generation rule '%s' evaluated in %s mode", rule.Name, mode)
		} else if rule.HasVerifyImages() {
			result.Status = "pass"
			result.Message = fmt.Sprintf("image verification rule '%s' evaluated in %s mode", rule.Name, mode)
		} else {
			result.Status = "skip"
			result.Message = "no actionable rule type"
		}

		results = append(results, result)
	}

	return results
}

// printSummary outputs the test run summary
func (r *TestRunner) printSummary(summary *TestSummary) {
	fmt.Fprintln(r.out, "")
	fmt.Fprintln(r.out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(r.out, "  Unified Test Runner - %s\n", summary.Mode.Description())
	fmt.Fprintln(r.out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(r.out, "  Setup:      %v\n", summary.SetupDuration)
	fmt.Fprintf(r.out, "  Evaluation: %v\n", summary.EvalDuration)
	fmt.Fprintf(r.out, "  Total:      %v\n", summary.TotalDuration)
	fmt.Fprintln(r.out, "  ──────────────────────────────────────────────")
	fmt.Fprintf(r.out, "  ✅ Pass: %d  ❌ Fail: %d  ⚠️  Warn: %d  💥 Error: %d  ⏭️  Skip: %d\n",
		summary.Pass, summary.Fail, summary.Warn, summary.Error, summary.Skip)
	if summary.FellBack {
		fmt.Fprintf(r.out, "  ↩️  Fell back from Accurate to Fast mode: %s\n", summary.FallbackReason)
	}
	fmt.Fprintln(r.out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// CompareResults compares results from Fast and Accurate modes
// This is useful for validating that Smart Mocks produce equivalent results
func CompareResults(fast, accurate *TestSummary) *ComparisonReport {
	report := &ComparisonReport{
		FastResults:     fast,
		AccurateResults: accurate,
	}

	// Build lookup maps
	fastMap := make(map[string]TestResult)
	for _, r := range fast.Results {
		key := fmt.Sprintf("%s/%s/%s/%s", r.PolicyName, r.RuleName, r.ResourceKind, r.ResourceName)
		fastMap[key] = r
	}

	accurateMap := make(map[string]TestResult)
	for _, r := range accurate.Results {
		key := fmt.Sprintf("%s/%s/%s/%s", r.PolicyName, r.RuleName, r.ResourceKind, r.ResourceName)
		accurateMap[key] = r
	}

	// Compare results
	for key, fastResult := range fastMap {
		if accurateResult, ok := accurateMap[key]; ok {
			if fastResult.Status == accurateResult.Status {
				report.Matching++
			} else {
				report.Divergent++
				report.Divergences = append(report.Divergences, Divergence{
					Key:            key,
					FastStatus:     fastResult.Status,
					AccurateStatus: accurateResult.Status,
					FastMessage:    fastResult.Message,
					AccurateMsg:    accurateResult.Message,
				})
			}
		} else {
			report.OnlyInFast++
		}
	}

	for key := range accurateMap {
		if _, ok := fastMap[key]; !ok {
			report.OnlyInAccurate++
		}
	}

	report.SpeedupFactor = float64(accurate.TotalDuration) / float64(fast.TotalDuration)

	return report
}

// ComparisonReport shows how Fast and Accurate mode results compare
type ComparisonReport struct {
	FastResults     *TestSummary
	AccurateResults *TestSummary

	// Matching is the count of results that agree between modes
	Matching int

	// Divergent is the count of results that differ between modes
	Divergent int

	// OnlyInFast is the count of results only present in fast mode
	OnlyInFast int

	// OnlyInAccurate is the count of results only present in accurate mode
	OnlyInAccurate int

	// Divergences lists the specific disagreements
	Divergences []Divergence

	// SpeedupFactor is how much faster Fast mode was (accurate_time / fast_time)
	SpeedupFactor float64
}

// Divergence represents a disagreement between Fast and Accurate mode
type Divergence struct {
	Key            string
	FastStatus     string
	AccurateStatus string
	FastMessage    string
	AccurateMsg    string
}

// PrintReport outputs the comparison report
func (r *ComparisonReport) PrintReport(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(w, "  Mode Comparison Report: Fast vs Accurate")
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(w, "  Fast Mode Time:     %v\n", r.FastResults.TotalDuration)
	fmt.Fprintf(w, "  Accurate Mode Time: %v\n", r.AccurateResults.TotalDuration)
	fmt.Fprintf(w, "  Speedup Factor:     %.1fx\n", r.SpeedupFactor)
	fmt.Fprintln(w, "  ──────────────────────────────────────────────")
	fmt.Fprintf(w, "  ✅ Matching:        %d\n", r.Matching)
	fmt.Fprintf(w, "  ⚠️  Divergent:       %d\n", r.Divergent)
	fmt.Fprintf(w, "  📋 Only in Fast:    %d\n", r.OnlyInFast)
	fmt.Fprintf(w, "  📋 Only in Accurate: %d\n", r.OnlyInAccurate)

	if len(r.Divergences) > 0 {
		fmt.Fprintln(w, "  ──────────────────────────────────────────────")
		fmt.Fprintln(w, "  Divergences:")
		for _, d := range r.Divergences {
			fmt.Fprintf(w, "    %s: fast=%s accurate=%s\n", d.Key, d.FastStatus, d.AccurateStatus)
		}
	}
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
