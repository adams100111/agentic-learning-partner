package domain

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(value string) (Version, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.Split(value, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return Version{}, fmt.Errorf("invalid semantic version %q", value)
	}
	numbers := make([]int, 3)
	for i, part := range parts {
		if part == "" {
			return Version{}, fmt.Errorf("invalid semantic version %q", value)
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("invalid semantic version %q", value)
		}
		numbers[i] = n
	}
	return Version{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}, nil
}

func (v Version) Compare(other Version) int {
	switch {
	case v.Major < other.Major:
		return -1
	case v.Major > other.Major:
		return 1
	case v.Minor < other.Minor:
		return -1
	case v.Minor > other.Minor:
		return 1
	case v.Patch < other.Patch:
		return -1
	case v.Patch > other.Patch:
		return 1
	default:
		return 0
	}
}

func Satisfies(version, expression string) (bool, error) {
	current, err := ParseVersion(version)
	if err != nil {
		return false, err
	}
	terms := strings.Fields(expression)
	if len(terms) == 0 {
		return false, fmt.Errorf("version range is empty")
	}

	for _, term := range terms {
		operator, raw := splitComparator(term)
		target, err := ParseVersion(raw)
		if err != nil {
			return false, fmt.Errorf("invalid range %q: %w", expression, err)
		}
		comparison := current.Compare(target)
		ok := false
		switch operator {
		case ">":
			ok = comparison > 0
		case ">=":
			ok = comparison >= 0
		case "<":
			ok = comparison < 0
		case "<=":
			ok = comparison <= 0
		case "=", "":
			ok = comparison == 0
		default:
			return false, fmt.Errorf("unsupported version comparator %q", operator)
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func splitComparator(term string) (string, string) {
	for _, operator := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(term, operator) {
			return operator, strings.TrimPrefix(term, operator)
		}
	}
	return "", term
}
