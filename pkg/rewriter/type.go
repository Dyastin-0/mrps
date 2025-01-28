package rewriter

type RewriteType string

const (
	PrefixRewrite RewriteType = "prefix"
	RegexRewrite  RewriteType = "regex"
)

type RewriteRule struct {
	Type       RewriteType `yaml:"type"`
	Value      string      `yaml:"value"`
	ReplaceVal string      `yaml:"replace_val"`
}

type Rewriter struct {
	rules RewriteRule
}
