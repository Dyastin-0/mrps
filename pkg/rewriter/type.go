package rewriter

type RewriteType string

const (
	PrefixRewrite RewriteType = "prefix"
	RegexRewrite  RewriteType = "regex"
)

type RewriteRule struct {
	Type       RewriteType `json:"type"`
	Value      string      `json:"value"`
	ReplaceVal string      `json:"replace_val"`
}

type Rewriter struct {
	rules RewriteRule
}
