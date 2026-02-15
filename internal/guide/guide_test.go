package guide

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakeGuideClient struct{}

func (f *fakeGuideClient) Get(_ context.Context, path string) ([]byte, error) {
	if path == styleURL {
		return []byte("# Terraform Style Guide\n\nBe consistent."), nil
	}
	if strings.HasPrefix(path, moduleDevBase) {
		section := strings.TrimPrefix(path, moduleDevBase+"/")
		section = strings.TrimSuffix(section, ".mdx")
		return []byte(fmt.Sprintf("# %s\n\nContent for %s.", section, section)), nil
	}
	return nil, fmt.Errorf("unexpected Get path: %s", path)
}

func TestFetchStyleGuide(t *testing.T) {
	content, err := FetchStyleGuide(context.Background(), &fakeGuideClient{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content, "Style Guide") {
		t.Errorf("expected style guide content, got: %s", content)
	}
}

func TestFetchModuleDevGuide_SingleSection(t *testing.T) {
	content, err := FetchModuleDevGuide(context.Background(), &fakeGuideClient{}, "composition")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content, "composition") {
		t.Errorf("expected composition content, got: %s", content)
	}
}

func TestFetchModuleDevGuide_AllSections(t *testing.T) {
	content, err := FetchModuleDevGuide(context.Background(), &fakeGuideClient{}, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, section := range ModuleDevSections {
		if !strings.Contains(content, section) {
			t.Errorf("expected section %s in all output", section)
		}
	}
	// All sections should be joined with separator
	if !strings.Contains(content, "---") {
		t.Error("expected section separator in all output")
	}
}

func TestFetchModuleDevGuide_DefaultAll(t *testing.T) {
	content, err := FetchModuleDevGuide(context.Background(), &fakeGuideClient{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty string should default to "all"
	for _, section := range ModuleDevSections {
		if !strings.Contains(content, section) {
			t.Errorf("expected section %s in default output", section)
		}
	}
}

func TestFetchModuleDevGuide_InvalidSection(t *testing.T) {
	_, err := FetchModuleDevGuide(context.Background(), &fakeGuideClient{}, "invalid")
	if err == nil {
		t.Fatal("expected error for invalid section")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
