package xmlconfig

import (
	"bytes"
	"cmp"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// Update identifies one XML target and its resolved replacement text.
type Update struct {
	// ID identifies the update in diagnostics.
	ID string
	// Path selects one existing element or attribute.
	Path string
	// Value contains replacement text, never raw XML.
	Value string
}

// byteRange locates a half-open region of the original XML file.
type byteRange struct {
	// start is the first byte included in the region.
	start int64
	// end is the first byte after the region.
	end int64
}

// documentAttribute connects a decoded attribute to its original value span.
type documentAttribute struct {
	// name retains the qualified attribute name from the source.
	name string
	// value is the decoded attribute value.
	value string
	// region locates the value between its original quotes.
	region byteRange
}

// replacement describes an escaped edit to one original source region.
type replacement struct {
	// region locates the bytes replaced by this edit.
	region byteRange
	// value contains the replacement bytes.
	value []byte
}

// selection contains a compiled update and its match count.
type selection struct {
	// update describes the requested target and value.
	update Update
	// steps holds the element selectors from root to target.
	steps []pathStep
	// attribute names the target attribute when this is not a text update.
	attribute string
	// escaped contains XML-escaped replacement text.
	escaped []byte
	// matches counts selected elements in the original document.
	matches int
}

// scanFrame stores only the state needed for the currently open XML element.
type scanFrame struct {
	// name retains the source-qualified name for expanding empty elements.
	name string
	// opening locates the opening tag.
	opening byteRange
	// selfClosing identifies the empty-element syntax.
	selfClosing bool
	// prefixes records which update paths match through this element.
	prefixes []bool
	// positions counts siblings only for indexed selector names.
	positions map[string]int
	// textSelection identifies the last update selecting this element's text, or -1.
	textSelection int
	// textRanges locates text chunks of a selected leaf, preserving intervening comments.
	textRanges []byteRange
	// textPosition counts decoded text bytes compared with the replacement value.
	textPosition int
	// equal reports whether decoded text has matched the replacement so far.
	equal bool
	// hasChildren prevents replacing a subtree with text.
	hasChildren bool
}

// planEdits scans and validates XML while retaining only open elements and selected edits.
func planEdits(
	source io.ReaderAt,
	size int64,
	updates []Update,
) ([]replacement, error) {
	selections, indexedNames, err := compileSelections(updates)
	if err != nil {
		return nil, err
	}
	offset := xmlContentOffset(source)
	decoder := xml.NewDecoder(io.NewSectionReader(source, offset, size-offset))
	edits := make(map[byteRange]replacement)
	var stack []*scanFrame
	roots := 0

	for {
		before := offset + decoder.InputOffset()
		token, err := decoder.Token()
		after := offset + decoder.InputOffset()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid XML document at line %d", xmlErrorLine(err))
		}

		switch token := token.(type) {
		case xml.StartElement:
			if len(stack) == 0 {
				roots++
				if roots != 1 {
					return nil, fmt.Errorf("XML document must have exactly one root")
				}
			} else {
				stack[len(stack)-1].hasChildren = true
			}
			frame, err := startFrame(source, byteRange{before, after}, token, stack, selections, indexedNames, edits)
			if err != nil {
				return nil, err
			}
			stack = append(stack, frame)
		case xml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(token)) != "" {
					return nil, fmt.Errorf("unexpected text outside XML root")
				}
			} else {
				compareText(stack[len(stack)-1], selections, token, byteRange{before, after})
			}
		case xml.EndElement:
			frame := stack[len(stack)-1]
			if err := finishFrame(frame, selections, edits); err != nil {
				return nil, err
			}
			stack = stack[:len(stack)-1]
		}
	}

	if roots == 0 {
		return nil, fmt.Errorf("XML document has no root")
	}
	for _, selection := range selections {
		if selection.matches == 0 {
			return nil, fmt.Errorf("update %s: path matched no elements", selection.update.ID)
		}
	}

	result := make([]replacement, 0, len(edits))
	for _, edit := range edits {
		result = append(result, edit)
	}
	slices.SortFunc(result, compareReplacementOffsets)

	return result, nil
}

// compileSelections validates paths and values and identifies the sibling counters needed.
func compileSelections(updates []Update) ([]selection, map[string]bool, error) {
	indexed := make(map[string]bool)
	var selections []selection

	for _, update := range updates {
		steps, attribute, err := parseSelector(update.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("update %s: %w", update.ID, err)
		}
		if !validXMLText(update.Value) {
			return nil, nil, fmt.Errorf("update %s contains invalid XML text", update.ID)
		}

		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(update.Value)); err != nil {
			return nil, nil, err
		}
		selections = append(
			selections,
			selection{
				update:    update,
				steps:     steps,
				attribute: attribute,
				escaped:   escaped.Bytes(),
			})
		for _, step := range steps {
			if step.index > 0 {
				indexed[step.name] = true
			}
		}
	}

	return selections, indexed, nil
}

// startFrame matches an opening element and records attribute replacements.
func startFrame(
	source io.ReaderAt,
	opening byteRange,
	token xml.StartElement,
	stack []*scanFrame,
	selections []selection,
	indexed map[string]bool,
	edits map[byteRange]replacement,
) (*scanFrame, error) {
	if !uniqueAttributes(token.Attr) {
		return nil, fmt.Errorf("XML element has duplicate attributes")
	}
	raw := make([]byte, opening.end-opening.start)
	if _, err := source.ReadAt(raw, opening.start); err != nil {
		return nil, err
	}
	name, attributes := openingMetadata(string(raw), opening, token.Attr)
	frame := &scanFrame{
		name:          name,
		opening:       opening,
		selfClosing:   bytes.HasSuffix(raw, []byte("/>")),
		prefixes:      make([]bool, len(selections)),
		textSelection: -1,
		equal:         true,
	}
	if len(indexed) != 0 {
		frame.positions = make(map[string]int)
	}

	var parent *scanFrame
	if len(stack) != 0 {
		parent = stack[len(stack)-1]
		if indexed[name] {
			parent.positions[name]++
		}
		if indexed["*"] {
			parent.positions["*"]++
		}
	}

	depth := len(stack)
	for index := range selections {
		selection := &selections[index]
		if !selection.canMatch(depth, parent, index) {
			continue
		}
		step := selection.steps[depth]
		position := step.siblingPosition(parent)
		if !step.matches(name, attributes, position) {
			continue
		}
		frame.prefixes[index] = true
		if depth != len(selection.steps)-1 {
			continue
		}
		selection.matches++
		if selection.matches > 1 {
			return nil, fmt.Errorf("update %s: path matched multiple elements", selection.update.ID)
		}
		if selection.attribute == "" {
			frame.textSelection = index
			continue
		}
		if err := selectAttribute(attributes, selection, edits); err != nil {
			return nil, err
		}
	}

	return frame, nil
}

// selectAttribute records the final value for an existing selected attribute.
func selectAttribute(
	attributes []documentAttribute,
	selection *selection,
	edits map[byteRange]replacement,
) error {
	for _, attribute := range attributes {
		if attribute.name != selection.attribute {
			continue
		}
		delete(edits, attribute.region)
		if attribute.value != selection.update.Value {
			edits[attribute.region] = replacement{region: attribute.region, value: selection.escaped}
		}
		return nil
	}

	return fmt.Errorf("update %s: attribute does not exist", selection.update.ID)
}

// compareText compares decoded chunks without retaining the selected property's full value.
func compareText(
	frame *scanFrame,
	selections []selection,
	text []byte,
	region byteRange,
) {
	if frame.textSelection < 0 {
		return
	}

	value := selections[frame.textSelection].update.Value
	next := frame.textPosition + len(text)
	if !textChunkMatches(value, frame.textPosition, text) {
		frame.equal = false
	}

	frame.textPosition = next
	frame.textRanges = append(frame.textRanges, region)
}

// finishFrame records text edits only for the final update targeting a selected leaf.
func finishFrame(
	frame *scanFrame,
	selections []selection,
	edits map[byteRange]replacement,
) error {
	if frame.textSelection < 0 {
		return nil
	}

	selection := selections[frame.textSelection]
	if frame.hasChildren {
		return fmt.Errorf("update %s: text replacement requires a leaf element", selection.update.ID)
	}
	if frame.textEquals(selection.update.Value) {
		return nil
	}

	if frame.selfClosing {
		region := byteRange{frame.opening.end - 2, frame.opening.end}
		edits[region] = replacement{region: region, value: []byte(">" + string(selection.escaped) + "</" + frame.name + ">")}
		return nil
	}
	if len(frame.textRanges) == 0 {
		region := byteRange{frame.opening.end, frame.opening.end}
		edits[region] = replacement{region: region, value: selection.escaped}
		return nil
	}

	for index, region := range frame.textRanges {
		var value []byte
		if index == 0 {
			value = selection.escaped
		}
		edits[region] = replacement{region: region, value: value}
	}
	return nil
}

// xmlErrorLine returns a safe syntax-error location without exposing property contents.
func xmlErrorLine(err error) int {
	var syntax *xml.SyntaxError
	if errors.As(err, &syntax) {
		return syntax.Line
	}
	return 0
}

// xmlContentOffset skips an optional UTF-8 byte-order mark before XML decoding.
func xmlContentOffset(source io.ReaderAt) int64 {
	var prefix [3]byte
	if _, err := source.ReadAt(prefix[:], 0); err != nil {
		return 0
	}
	if !bytes.Equal(prefix[:], []byte{0xEF, 0xBB, 0xBF}) {
		return 0
	}
	return int64(len(prefix))
}

// compareReplacementOffsets orders edits by their positions in the original document.
func compareReplacementOffsets(left, right replacement) int {
	return cmp.Compare(left.region.start, right.region.start)
}

// canMatch checks that a path reaches this depth through a matching parent.
func (selection selection) canMatch(depth int, parent *scanFrame, index int) bool {
	if depth >= len(selection.steps) {
		return false
	}
	if parent == nil {
		return true
	}
	return parent.prefixes[index]
}

// siblingPosition returns the counter needed by an indexed selector step.
func (step pathStep) siblingPosition(parent *scanFrame) int {
	if parent == nil {
		return 1
	}
	if step.index == 0 {
		return 1
	}
	return parent.positions[step.name]
}

// textChunkMatches compares a decoded chunk without slicing beyond the replacement value.
func textChunkMatches(value string, position int, text []byte) bool {
	if position > len(value) {
		return false
	}
	return strings.HasPrefix(value[position:], string(text))
}

// textEquals reports whether every decoded text chunk matched the complete replacement.
func (frame *scanFrame) textEquals(value string) bool {
	return frame.equal && frame.textPosition == len(value)
}
