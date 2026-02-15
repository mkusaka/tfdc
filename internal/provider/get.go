package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GetResult holds the result of fetching a single provider doc.
type GetResult struct {
	ID          string
	Content     string
	ContentType string
}

// GetDoc fetches a single provider doc by numeric ID.
func GetDoc(ctx context.Context, client APIClient, docID string) (*GetResult, error) {
	docID = strings.TrimSpace(docID)
	if docID == "" {
		return nil, &ValidationError{Message: "-doc-id is required"}
	}
	if _, err := strconv.Atoi(docID); err != nil {
		return nil, &ValidationError{Message: fmt.Sprintf("-doc-id must be numeric: %s", docID)}
	}

	detail, _, err := getProviderDocDetail(ctx, client, docID, false)
	if err != nil {
		return nil, err
	}

	return &GetResult{
		ID:          detail.Data.ID,
		Content:     detail.Data.Attributes.Content,
		ContentType: "text/markdown",
	}, nil
}
