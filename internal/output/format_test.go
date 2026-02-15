package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteSearch_JSON(t *testing.T) {
	items := []map[string]any{
		{"id": "1", "title": "foo"},
		{"id": "2", "title": "bar"},
	}
	var buf bytes.Buffer
	if err := WriteSearch(&buf, "json", items, 2, []string{"id", "title"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result SearchResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
}

func TestWriteSearch_Text(t *testing.T) {
	items := []map[string]any{
		{"id": "1", "name": "vpc"},
	}
	var buf bytes.Buffer
	if err := WriteSearch(&buf, "text", items, 1, []string{"id", "name"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "id") || !strings.Contains(out, "name") {
		t.Fatalf("expected header columns, got: %s", out)
	}
	if !strings.Contains(out, "vpc") {
		t.Fatalf("expected data row, got: %s", out)
	}
}

func TestWriteSearch_Markdown(t *testing.T) {
	items := []map[string]any{
		{"id": "1", "name": "vpc"},
	}
	var buf bytes.Buffer
	if err := WriteSearch(&buf, "markdown", items, 1, []string{"id", "name"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "| id | name |") {
		t.Fatalf("expected markdown header, got: %s", out)
	}
	if !strings.Contains(out, "| --- | --- |") {
		t.Fatalf("expected markdown separator, got: %s", out)
	}
}

func TestWriteDetail_JSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDetail(&buf, "json", "123", "content here", "text/markdown"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result DetailResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if result.ID != "123" {
		t.Fatalf("expected id=123, got %s", result.ID)
	}
	if result.Content != "content here" {
		t.Fatalf("expected content='content here', got %s", result.Content)
	}
}

func TestWriteDetail_Text(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDetail(&buf, "text", "123", "raw content", "text/markdown"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "raw content" {
		t.Fatalf("expected raw content, got: %s", buf.String())
	}
}

func TestWriteDetail_Markdown(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDetail(&buf, "markdown", "123", "# Title\nbody", "text/markdown"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "# Title\nbody" {
		t.Fatalf("expected markdown content, got: %s", buf.String())
	}
}

func TestWriteSearch_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := WriteSearch(&buf, "xml", nil, 0, nil)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestWriteDetail_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := WriteDetail(&buf, "xml", "", "", "")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}
