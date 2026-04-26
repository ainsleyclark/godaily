package news

// Source defines a provider or source of information.
type Source string

// Source constants
const (
	SourceDevTo  Source = "dev_to"
	SourceGoBlog Source = "go_blog"
	SourceGitHub Source = "github"
	SourceReddit Source = "reddit"
	SourceHN     Source = "hacker_news"
)

// Sources defines a list of all source types.
var Sources = []Source{
	SourceDevTo,
	SourceGoBlog,
	SourceGitHub,
	SourceReddit,
	SourceHN,
}

// String implements fmt.Stringer on source.
func (s Source) String() string {
	return string(s)
}
