package rewriter

type RewriteType string

const (
	PrefixRewrite RewriteType = "prefix"
	RegexRewrite  RewriteType = "regex"
)

type RewriteRule struct {
	Type       RewriteType `yaml:"type,omitempty"`
	Value      string      `yaml:"value,omitempty"`
	ReplaceVal string      `yaml:"replace_val,omitempty"`
}

type Rewriter struct {
	rules RewriteRule
}
