package checks

import "fmt"

// Severity distinguishes a hard failure from an advisory warning.
type Severity int

const (
	Error Severity = iota
	Warning
)

// Finding is one validation result. Rule is a short stable identifier
// (e.g. "closed-enum", "flow-graph") so tooling and tests can key off it
// without parsing prose.
type Finding struct {
	Severity Severity
	Rule     string
	Message  string
}

func (f Finding) String() string {
	sev := "ERROR"
	if f.Severity == Warning {
		sev = "WARN"
	}
	return fmt.Sprintf("[%s] %s: %s", sev, f.Rule, f.Message)
}

func errf(rule, format string, args ...interface{}) Finding {
	return Finding{Severity: Error, Rule: rule, Message: fmt.Sprintf(format, args...)}
}

func warnf(rule, format string, args ...interface{}) Finding {
	return Finding{Severity: Warning, Rule: rule, Message: fmt.Sprintf(format, args...)}
}

// HasErrors reports whether any finding in the list is an Error.
func HasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == Error {
			return true
		}
	}
	return false
}
