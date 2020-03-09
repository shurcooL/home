package issues

// IssueListOptions are options for list operations.
type IssueListOptions struct {
	State StateFilter
}

// StateFilter is a filter by state.
type StateFilter State

const (
	// AllStates is a state filter that includes all issues.
	AllStates StateFilter = "all"
)

// ListOptions controls pagination.
type ListOptions struct {
	// Start is the index of first result to retrieve, zero-indexed.
	Start int

	// Length is the number of results to include.
	Length int
}
