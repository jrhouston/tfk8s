// NOTE this file was lifted verbatim from internal/repl in the terraform project
// because the FormatValue function became internal in v1.0.0

// NOTE this file has since been modified so it has drifted from what was in
// terraform core

package terraform

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

// FormatValue formats a value in a way that resembles Terraform language syntax
// and uses the type conversion functions where necessary to indicate exactly
// what type it is given, so that equality test failures can be quickly
// understood.
func FormatValue(v cty.Value, indent int, stripKeyQuotes bool) string {
	if !v.IsKnown() {
		return "(known after apply)"
	}
	if v.IsMarked() {
		return "(sensitive)"
	}
	if v.IsNull() {
		ty := v.Type()
		switch {
		case ty == cty.DynamicPseudoType:
			return "null"
		case ty == cty.String:
			return "tostring(null)"
		case ty == cty.Number:
			return "tonumber(null)"
		case ty == cty.Bool:
			return "tobool(null)"
		case ty.IsListType():
			return fmt.Sprintf("tolist(null) /* of %s */", ty.ElementType().FriendlyName())
		case ty.IsSetType():
			return fmt.Sprintf("toset(null) /* of %s */", ty.ElementType().FriendlyName())
		case ty.IsMapType():
			return fmt.Sprintf("tomap(null) /* of %s */", ty.ElementType().FriendlyName())
		default:
			return fmt.Sprintf("null /* %s */", ty.FriendlyName())
		}
	}

	ty := v.Type()
	switch {
	case ty.IsPrimitiveType():
		switch ty {
		case cty.String:
			if formatted, isMultiline := formatMultilineString(v, indent); isMultiline {
				return formatted
			}
			return strconv.Quote(v.AsString())
		case cty.Number:
			bf := v.AsBigFloat()
			return bf.Text('f', -1)
		case cty.Bool:
			if v.True() {
				return "true"
			} else {
				return "false"
			}
		}
	case ty.IsObjectType():
		return formatMappingValue(v, indent, stripKeyQuotes)
	case ty.IsTupleType():
		return formatSequenceValue(v, indent, stripKeyQuotes)
	case ty.IsListType():
		return fmt.Sprintf("tolist(%s)", formatSequenceValue(v, indent, stripKeyQuotes))
	case ty.IsSetType():
		return fmt.Sprintf("toset(%s)", formatSequenceValue(v, indent, stripKeyQuotes))
	case ty.IsMapType():
		return fmt.Sprintf("tomap(%s)", formatMappingValue(v, indent, stripKeyQuotes))
	}

	// Should never get here because there are no other types
	return fmt.Sprintf("%#v", v)
}

// defaultDelimiter is "End Of Text" by convention
const defaultDelimiter = "EOT"

func formatMultilineString(v cty.Value, indent int) (string, bool) {
	str := v.AsString()
	lines := strings.Split(str, "\n")
	if len(lines) < 2 {
		return "", false
	}

	// If the value is indented, we use the indented form of heredoc for readability.
	operator := "<<"
	if indent > 0 {
		operator = "<<-"
	}

	delimiter := defaultDelimiter

OUTER:
	for {
		// Check if any of the lines are in conflict with the delimiter. The
		// parser allows leading and trailing whitespace, so we must remove it
		// before comparison.
		for _, line := range lines {
			// If the delimiter matches a line, extend it and start again
			if strings.TrimSpace(line) == delimiter {
				delimiter = delimiter + "_"
				continue OUTER
			}
		}

		// None of the lines match the delimiter, so we're ready
		break
	}

	// Write the heredoc, with indentation as appropriate.
	var buf strings.Builder

	buf.WriteString(operator)
	buf.WriteString(delimiter)
	for _, line := range lines {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(line)
	}
	buf.WriteByte('\n')
	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteString(delimiter)

	return buf.String(), true
}

func formatMappingValue(v cty.Value, indent int, stripKeyQuotes bool) string {
	var buf strings.Builder
	count := 0
	buf.WriteByte('{')
	indent += 2
	for it := v.ElementIterator(); it.Next(); {
		count++
		k, v := it.Element()
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		key := FormatValue(k, indent, stripKeyQuotes)
		if stripKeyQuotes {
			// they can be unquoted if it starts with a letter
			// and only contains alphanumeric characeters, dashes, and underlines
			m := regexp.MustCompile(`^"[A-Za-z][0-9A-Za-z-_]+"$`)
			if m.MatchString(key) {
				key = key[1 : len(key)-1]
			}
		}
		buf.WriteString(key)
		buf.WriteString(" = ")
		buf.WriteString(FormatValue(v, indent, stripKeyQuotes))
	}
	indent -= 2
	if count > 0 {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
	}
	buf.WriteByte('}')
	return buf.String()
}

func formatSequenceValue(v cty.Value, indent int, stripKeyQuotes bool) string {
	var buf strings.Builder
	count := 0
	buf.WriteByte('[')
	indent += 2
	for it := v.ElementIterator(); it.Next(); {
		count++
		_, v := it.Element()
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		formattedValue := FormatValue(v, indent, stripKeyQuotes)
		buf.WriteString(formattedValue)
		if strings.HasSuffix(formattedValue, defaultDelimiter) {
			// write an additional newline if the value was a multiline string
			buf.WriteByte('\n')
			buf.WriteString(strings.Repeat(" ", indent))
		}
		buf.WriteByte(',')
	}
	indent -= 2
	if count > 0 {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
	}
	buf.WriteByte(']')
	return buf.String()
}
