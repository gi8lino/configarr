package xmlconfig

import (
	"encoding/xml"
	"slices"
	"strings"
	"unicode/utf8"
)

// openingMetadata associates decoder attributes with their original quoted value spans.
func openingMetadata(tag string, opening byteRange, decoded []xml.Attr) (string, []documentAttribute) {
	end := strings.IndexAny(tag, " \t\r\n/>")
	name := tag[1:end]
	cursor := end
	var attributes []documentAttribute

	for _, attribute := range decoded {
		cursor = skipXMLSpace(tag, cursor)
		nameStart := cursor
		for cursor < len(tag) && !isAttributeNameEnd(tag[cursor]) {
			cursor++
		}
		attributeName := tag[nameStart:cursor]
		cursor = skipXMLSpace(tag, cursor)
		cursor++ // The decoder has already validated the equals sign.
		cursor = skipXMLSpace(tag, cursor)

		quote := tag[cursor]
		cursor++
		valueStart := cursor
		for tag[cursor] != quote {
			cursor++
		}

		attributes = append(attributes, documentAttribute{
			name:  attributeName,
			value: attribute.Value,
			region: byteRange{
				start: opening.start + int64(valueStart),
				end:   opening.start + int64(cursor),
			},
		})
		cursor++
	}

	return name, attributes
}

// skipXMLSpace advances past the four whitespace characters recognized in XML tags.
func skipXMLSpace(text string, cursor int) int {
	for cursor < len(text) && isXMLSpace(text[cursor]) {
		cursor++
	}
	return cursor
}

// validXMLText checks XML 1.0 character ranges without replacing invalid input.
func validXMLText(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}

	for _, char := range value {
		if !isXMLCharacter(char) {
			return false
		}
	}
	return true
}

// isXMLCharacter recognizes XML 1.0 whitespace and the three permitted character ranges.
func isXMLCharacter(char rune) bool {
	switch {
	case slices.Contains([]rune{'\t', '\n', '\r'}, char):
		return true
	case runeInRange(char, 0x20, 0xD7FF):
		return true
	case runeInRange(char, 0xE000, 0xFFFD):
		return true
	case runeInRange(char, 0x10000, utf8.MaxRune):
		return true
	default:
		return false
	}
}

// runeInRange reports whether a character lies within an inclusive Unicode range.
func runeInRange(char, first, last rune) bool {
	return char >= first && char <= last
}

// uniqueAttributes rejects duplicate expanded names that the XML decoder otherwise accepts.
func uniqueAttributes(attributes []xml.Attr) bool {
	seen := make(map[xml.Name]bool, len(attributes))
	for _, attribute := range attributes {
		if seen[attribute.Name] {
			return false
		}
		seen[attribute.Name] = true
	}
	return true
}

// isXMLSpace identifies the four ASCII whitespace characters allowed in XML markup.
func isXMLSpace(char byte) bool {
	return slices.Contains([]byte{' ', '\t', '\r', '\n'}, char)
}

// isAttributeNameEnd identifies whitespace or the equals sign following an attribute name.
func isAttributeNameEnd(char byte) bool {
	return char == '=' || isXMLSpace(char)
}
