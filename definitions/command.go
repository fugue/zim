package definitions

// Command to execute within a rule
type Command struct {
	Kind       string
	Argument   string
	Attributes map[string]interface{}
}
