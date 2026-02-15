package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakeGetClient struct{}

func (f *fakeGetClient) GetJSON(_ context.Context, path string, dst any) error {
	return fmt.Errorf("unexpected GetJSON call: %s", path)
}

func (f *fakeGetClient) Get(_ context.Context, path string) ([]byte, error) {
	if path == "/v2/provider-docs/8894603" {
		return []byte(`{"data":{"id":"8894603","attributes":{"category":"resources","slug":"aws_instance","title":"aws_instance","content":"# AWS Instance\n\nManage an EC2 instance."}}}`), nil
	}
	return nil, fmt.Errorf("unexpected Get call: %s", path)
}

func TestGetDoc_Success(t *testing.T) {
	result, err := GetDoc(context.Background(), &fakeGetClient{}, "8894603")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "8894603" {
		t.Errorf("expected id=8894603, got %s", result.ID)
	}
	if !strings.Contains(result.Content, "AWS Instance") {
		t.Errorf("expected content to contain 'AWS Instance', got: %s", result.Content)
	}
	if result.ContentType != "text/markdown" {
		t.Errorf("expected content_type=text/markdown, got %s", result.ContentType)
	}
}

func TestGetDoc_EmptyDocID(t *testing.T) {
	_, err := GetDoc(context.Background(), &fakeGetClient{}, "")
	if err == nil {
		t.Fatal("expected error for empty doc-id")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetDoc_NonNumericDocID(t *testing.T) {
	_, err := GetDoc(context.Background(), &fakeGetClient{}, "abc")
	if err == nil {
		t.Fatal("expected error for non-numeric doc-id")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "numeric") {
		t.Errorf("expected error about numeric, got: %v", err)
	}
}
