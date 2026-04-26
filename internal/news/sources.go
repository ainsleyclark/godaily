package news

// Source defines a provider or source of information.
type Source string

// Source constants
const (
	SourceDevTo  Source = "dev_to"
	SourceGitHub Source = "github"
	SourceReddit Source = "reddit"
)

// Sources defines a list of all source types.
var Sources = []Source{
	SourceDevTo,
	SourceGitHub,
	SourceReddit,
}

// String implements fmt.Stringer on source.
func (s Source) String() string {
	return string(s)
}
