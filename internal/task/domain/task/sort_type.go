package task

type SortType string

const (
	SortTypeTargetAt SortType = "target_at"
)

func NewSortType(s string) (SortType, error) {
	switch s {
	case string(SortTypeTargetAt):
		return SortType(s), nil
	default:
		return "", ErrInvalidSortType
	}
}

func (s SortType) OrderQuery() (string, error) {
	switch s {
	case SortTypeTargetAt:
		return "target_at ASC, created_at DESC", nil
	default:
		return "", ErrInvalidSortType
	}
}
