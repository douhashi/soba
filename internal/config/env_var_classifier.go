package config

// EnvVarCategory represents the category of an environment variable
type EnvVarCategory int

const (
	// SystemVariable represents system-managed variables that should never warn
	SystemVariable EnvVarCategory = iota
	// ConditionalVariable represents variables that warn based on configuration
	ConditionalVariable
	// UserVariable represents user-defined variables that always warn when undefined
	UserVariable
)

// String returns the string representation of the category
func (c EnvVarCategory) String() string {
	switch c {
	case SystemVariable:
		return "system"
	case ConditionalVariable:
		return "conditional"
	case UserVariable:
		return "user"
	default:
		return "unknown"
	}
}

// EnvVarClassifier is an interface for classifying environment variables
type EnvVarClassifier interface {
	// Classify determines the category of an environment variable
	Classify(envVar string, cfg *Config) EnvVarCategory
	// ShouldWarn determines if a warning should be shown for a missing environment variable
	ShouldWarn(envVar string, cfg *Config) bool
}

// defaultEnvVarClassifier is the default implementation of EnvVarClassifier
type defaultEnvVarClassifier struct{}

// NewEnvVarClassifier creates a new EnvVarClassifier instance
func NewEnvVarClassifier() EnvVarClassifier {
	return &defaultEnvVarClassifier{}
}

// Classify determines the category of an environment variable
func (c *defaultEnvVarClassifier) Classify(envVar string, cfg *Config) EnvVarCategory {
	if c.isSystemVariable(envVar) {
		return SystemVariable
	}
	if c.isConditionalVariable(envVar) {
		return ConditionalVariable
	}
	return UserVariable
}

// ShouldWarn determines if a warning should be shown for a missing environment variable
func (c *defaultEnvVarClassifier) ShouldWarn(envVar string, cfg *Config) bool {
	category := c.Classify(envVar, cfg)

	switch category {
	case SystemVariable:
		// System variables never warn
		return false
	case ConditionalVariable:
		// Conditional variables warn based on configuration
		return c.shouldWarnForConditionalVariable(envVar, cfg)
	case UserVariable:
		// User variables always warn
		return true
	default:
		// Unknown category defaults to warning
		return true
	}
}

// isSystemVariable checks if an environment variable is a system variable
func (c *defaultEnvVarClassifier) isSystemVariable(envVar string) bool {
	// Currently only PID is treated as a system variable
	// It's replaced at daemon startup, not during config load
	return envVar == "PID"
}

// isConditionalVariable checks if an environment variable is a conditional variable
func (c *defaultEnvVarClassifier) isConditionalVariable(envVar string) bool {
	// These variables have conditional warning logic based on configuration
	switch envVar {
	case "GITHUB_TOKEN", "SLACK_WEBHOOK_URL":
		return true
	default:
		return false
	}
}

// shouldWarnForConditionalVariable determines if a conditional variable should warn
func (c *defaultEnvVarClassifier) shouldWarnForConditionalVariable(envVar string, cfg *Config) bool {
	switch envVar {
	case "GITHUB_TOKEN":
		// Warn only when auth_method is "env"
		return cfg.GitHub.AuthMethod == "env"
	case "SLACK_WEBHOOK_URL":
		// Warn only when notifications_enabled is true
		return cfg.Slack.NotificationsEnabled
	default:
		// Unknown conditional variable defaults to no warning
		return false
	}
}
