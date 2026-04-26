package news

// Source defines a provider or source of information.
type Source string

// Source constants
const (
	SourceDevTo  Source = "dev_to"
	SourceGitHub Source = "github"
	SourceReddit Source = "reddit"
)
