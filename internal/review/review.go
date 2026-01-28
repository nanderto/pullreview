package review

// Review encapsulates the logic for preparing and posting code review comments.
type Review struct {
	PRID     string
	Diff     string
	Comments []Comment
	Summary  string
}

// Comment represents an inline or file-level comment to be posted on a PR.
type Comment struct {
	FilePath string
	Line     int
	Text     string
}

// NewReview creates a new Review instance.
func NewReview(prID, diff string) *Review {
	return &Review{
		PRID: prID,
		Diff: diff,
	}
}

// Placeholder for future review logic methods.
