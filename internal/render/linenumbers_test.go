package render

import (
	"testing"
)

func TestExtractASTBlockLines(t *testing.T) {
	src := []byte("# Header\n\nParagraph 1.\n\nParagraph 2.\n")
	blocks := ExtractASTBlockLines(src)
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if blocks[0] != 1 {
		t.Fatalf("expected header block at line 1, got %d", blocks[0])
	}
	if blocks[1] != 3 {
		t.Fatalf("expected paragraph 1 at line 3, got %d", blocks[1])
	}
	if blocks[2] != 5 {
		t.Fatalf("expected paragraph 2 at line 5, got %d", blocks[2])
	}
}
