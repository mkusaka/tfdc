package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// SearchResult is the JSON envelope for search commands.
type SearchResult struct {
	Items []map[string]any `json:"items"`
	Total int              `json:"total"`
}

// DetailResult is the JSON envelope for get/detail commands.
type DetailResult struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// FormatError indicates an unsupported output format.
type FormatError struct {
	Format string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("unsupported format: %s", e.Format)
}

// WriteSearch writes search results to w in the given format.
// columns controls the order and selection of fields for text/markdown output.
func WriteSearch(w io.Writer, format string, items []map[string]any, total int, columns []string) error {
	switch format {
	case "json":
		return writeJSON(w, SearchResult{Items: items, Total: total})
	case "text":
		return writeTable(w, items, columns)
	case "markdown":
		return writeMarkdownTable(w, items, columns)
	default:
		return &FormatError{Format: format}
	}
}

// WriteDetail writes a single detail/get result to w in the given format.
func WriteDetail(w io.Writer, format string, id, content, contentType string) error {
	switch format {
	case "json":
		return writeJSON(w, DetailResult{ID: id, Content: content, ContentType: contentType})
	case "text", "markdown":
		_, err := fmt.Fprint(w, content)
		return err
	default:
		return &FormatError{Format: format}
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeTable(w io.Writer, items []map[string]any, columns []string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, strings.Join(columns, "\t"))
	for _, item := range items {
		vals := make([]string, len(columns))
		for i, col := range columns {
			vals[i] = fmt.Sprintf("%v", item[col])
		}
		_, _ = fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	return tw.Flush()
}

func writeMarkdownTable(w io.Writer, items []map[string]any, columns []string) error {
	_, _ = fmt.Fprintf(w, "| %s |\n", strings.Join(columns, " | "))
	seps := make([]string, len(columns))
	for i := range seps {
		seps[i] = "---"
	}
	_, _ = fmt.Fprintf(w, "| %s |\n", strings.Join(seps, " | "))
	for _, item := range items {
		vals := make([]string, len(columns))
		for i, col := range columns {
			vals[i] = fmt.Sprintf("%v", item[col])
		}
		_, _ = fmt.Fprintf(w, "| %s |\n", strings.Join(vals, " | "))
	}
	return nil
}
