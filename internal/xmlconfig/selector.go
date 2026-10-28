package xmlconfig

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// pathStep selects a named element with an optional index or attribute predicate.
type pathStep struct {
	// name is the source-qualified element name to match.
	name string
	// index selects a one-based position among matching siblings when nonzero.
	index int
	// attribute filters matches by an existing attribute when nonempty.
	attribute string
	// value is the decoded attribute value required by the predicate.
	value string
}

// parseSelector compiles an absolute path into element steps and an optional attribute.
func parseSelector(path string) ([]pathStep, string, error) {
	parts, err := splitPath(path)
	if err != nil {
		return nil, "", err
	}

	attribute := ""
	if strings.HasPrefix(parts[len(parts)-1], "@") {
		attribute = strings.TrimPrefix(parts[len(parts)-1], "@")
		if !validPathName(attribute) {
			return nil, "", fmt.Errorf("invalid attribute name")
		}
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return nil, "", fmt.Errorf("path must select an element before an attribute")
	}

	var steps []pathStep
	for _, part := range parts {
		step, err := parsePathStep(part)
		if err != nil {
			return nil, "", err
		}
		steps = append(steps, step)
	}

	return steps, attribute, nil
}

// splitPath splits absolute child paths while preserving slashes inside quoted predicates.
func splitPath(path string) ([]string, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("path must be absolute")
	}

	var parts []string
	start, depth := 1, 0
	var quote rune

	for index, char := range path[1:] {
		offset := index + 1
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"':
			if depth != 1 {
				return nil, fmt.Errorf("unexpected quote in path")
			}
			quote = char
		case '[':
			depth++
			if depth != 1 {
				return nil, fmt.Errorf("nested predicates are unsupported")
			}
		case ']':
			depth--
			if depth != 0 {
				return nil, fmt.Errorf("unmatched predicate bracket")
			}
		case '/':
			if depth == 0 {
				parts = append(parts, path[start:offset])
				start = offset + 1
			}
		}
	}

	if !predicateIsClosed(depth, quote) {
		return nil, fmt.Errorf("unterminated path predicate")
	}
	parts = append(parts, path[start:])

	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("empty or descendant path steps are unsupported")
		}
	}

	return parts, nil
}

// parsePathStep validates named steps with one positional or attribute-equality predicate.
func parsePathStep(part string) (pathStep, error) {
	name, predicate, hasPredicate := strings.Cut(part, "[")
	step := pathStep{name: name}
	if !validElementSelectorName(name) {
		return step, fmt.Errorf("unsupported element name or path syntax")
	}
	if !hasPredicate {
		return step, nil
	}

	predicate, closed := strings.CutSuffix(predicate, "]")
	if !closed {
		return step, fmt.Errorf("invalid path predicate")
	}

	if position := positiveIndex(predicate); position != 0 {
		step.index = position
		return step, nil
	}

	attribute, value, valid := attributePredicateParts(predicate)
	if !valid {
		return step, fmt.Errorf("predicate must be a positive index or @attribute='value'")
	}

	step.attribute, step.value = attribute, value
	return step, nil
}

// isQuoted reports whether a predicate value uses one matching pair of quotes.
func isQuoted(value string) bool {
	if len(value) < 2 {
		return false
	}
	if !slices.Contains([]byte{'\'', '"'}, value[0]) {
		return false
	}
	if value[len(value)-1] != value[0] {
		return false
	}

	return !strings.ContainsRune(value[1:len(value)-1], rune(value[0]))
}

// validPathName accepts literal XML names and rejects XPath operators and wildcards.
func validPathName(name string) bool {
	if name == "" {
		return false
	}

	for index, char := range name {
		if index == 0 {
			if !validNameStart(char) {
				return false
			}
			continue
		}
		if !validNameContinuation(char) {
			return false
		}
	}
	return true
}

// validNameStart recognizes a letter, underscore, or namespace separator.
func validNameStart(char rune) bool {
	return unicode.IsLetter(char) || slices.Contains([]rune{'_', ':'}, char)
}

// validNameContinuation recognizes letters, digits, and XML name punctuation.
func validNameContinuation(char rune) bool {
	if validNameStart(char) {
		return true
	}
	return unicode.IsDigit(char) || slices.Contains([]rune{'-', '.'}, char)
}

// matches reports whether a streaming element satisfies one selector step.
func (step pathStep) matches(name string, attributes []documentAttribute, position int) bool {
	if !step.matchesName(name) {
		return false
	}
	if !step.matchesPosition(position) {
		return false
	}
	if step.attribute == "" {
		return true
	}

	for _, attribute := range attributes {
		if step.matchesAttribute(attribute) {
			return true
		}
	}
	return false
}

// predicateIsClosed reports whether both the bracket and any quoted value have ended.
func predicateIsClosed(depth int, quote rune) bool {
	return depth == 0 && quote == 0
}

// validElementSelectorName accepts a literal XML name or the element wildcard.
func validElementSelectorName(name string) bool {
	return name == "*" || validPathName(name)
}

// positiveIndex parses a one-based positional predicate and returns zero for other expressions.
func positiveIndex(predicate string) int {
	position, err := strconv.Atoi(predicate)
	if err != nil {
		return 0
	}
	if position < 1 {
		return 0
	}
	return position
}

// attributePredicateParts extracts a named attribute and its quoted literal value.
func attributePredicateParts(predicate string) (string, string, bool) {
	attribute, quoted, found := strings.Cut(predicate, "=")
	if !found {
		return "", "", false
	}
	name, prefixed := strings.CutPrefix(strings.TrimSpace(attribute), "@")
	if !prefixed {
		return "", "", false
	}
	if !validPathName(name) {
		return "", "", false
	}

	quoted = strings.TrimSpace(quoted)
	if !isQuoted(quoted) {
		return "", "", false
	}
	return name, quoted[1 : len(quoted)-1], true
}

// matchesName accepts the requested literal name or any name for a wildcard step.
func (step pathStep) matchesName(name string) bool {
	return step.name == "*" || step.name == name
}

// matchesPosition applies an optional one-based sibling index.
func (step pathStep) matchesPosition(position int) bool {
	return step.index == 0 || step.index == position
}

// matchesAttribute compares the predicate's attribute name and decoded value.
func (step pathStep) matchesAttribute(attribute documentAttribute) bool {
	return attribute.name == step.attribute && attribute.value == step.value
}
