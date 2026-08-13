// Package tokenizer provides a fast, pure-Go token counter.
//
// It embeds the OpenAI cl100k_base vocabulary and implements byte-level BPE
// scoring, so token counts are deterministic and require no network calls.
package tokenizer

import (
	"bytes"
	"embed"
	"encoding/base64"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

//go:embed data/cl100k_base.tiktoken
var vocabFS embed.FS

const vocabPath = "data/cl100k_base.tiktoken"

// Encoder is a ready-to-use byte-level BPE tokenizer for the cl100k_base
// vocabulary (GPT-4 family, claude token counting compatible at the UTF-8
// bytes level).
var Encoder *BPE

func init() {
	var err error
	Encoder, err = Load(vocabFS)
	if err != nil {
		// embed is fixed at build time; a failure here is a build bug.
		panic("tokenizer: " + err.Error())
	}
}

// BPE is a byte-level byte-pair encoding tokenizer.
type BPE struct {
	dict       map[string]int
	sorted     []string // vocabulary sorted by token string (bytewise)
	tokenBytes map[string][]byte
}

// Load builds a BPE from an embedded tiktoken vocabulary file.
//
// Each line is "<base64-token-bytes> <rank>". Tokens are decoded back to
// raw bytes so both lookup and BPE merges operate on the true token text.
func Load(fs embed.FS) (*BPE, error) {
	data, err := fs.ReadFile(vocabPath)
	if err != nil {
		return nil, err
	}
	b := &BPE{
		dict:       make(map[string]int),
		tokenBytes: make(map[string][]byte),
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			continue
		}
		rank, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		tok := string(decoded)
		b.dict[tok] = rank
		b.tokenBytes[tok] = decoded
	}
	for tok := range b.dict {
		b.sorted = append(b.sorted, tok)
	}
	sort.Slice(b.sorted, func(i, j int) bool { return b.dict[b.sorted[i]] < b.dict[b.sorted[j]] })
	return b, nil
}

// Encode returns the token IDs for s.
func (b *BPE) Encode(s string) []int {
	buf := append([]byte(nil), s...)
	if len(buf) == 0 {
		return nil
	}
	// Remove any UTF-8 BOM so byte positions match the reference tokenizer.
	buf = bytes.TrimPrefix(buf, []byte{0xEF, 0xBB, 0xBF})

	parts := splitOnRune(buf, ' ')

	// If the text doesn't end with a space, all but the last part are
	// word-boundaries and get a trailing space, matching the reference
	// tokenizer's byte-level split.
	if len(parts) > 1 && !endsWithSpace(buf) {
		for i := 0; i < len(parts)-1; i++ {
			parts[i] = append(parts[i], ' ')
		}
	}

	var ids []int
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		ids = append(ids, b.encodePiece(part)...)
	}
	return ids
}

// Count returns the number of tokens in s.
func (b *BPE) Count(s string) int {
	return len(b.Encode(s))
}

func (b *BPE) encodePiece(piece []byte) []int {
	if len(piece) == 1 {
		if id, ok := b.dict[string(piece)]; ok {
			return []int{id}
		}
		return nil
	}
	var parts [][]byte
	for i := 0; i < len(piece); {
		_, size := utf8.DecodeRune(piece[i:])
		parts = append(parts, piece[i:i+size])
		i += size
	}

	// Repeatedly merge the pair with the lowest rank, like the reference BPE.
	for {
		bestI, bestRank := -1, 0
		for i := 0; i < len(parts)-1; i++ {
			key := string(parts[i]) + string(parts[i+1])
			if rank, ok := b.dict[key]; ok {
				if bestI < 0 || rank < bestRank {
					bestI, bestRank = i, rank
				}
			}
		}
		if bestI < 0 {
			break
		}
		merged := append(parts[bestI], parts[bestI+1]...)
		parts = append(parts[:bestI], append([][]byte{merged}, parts[bestI+2:]...)...)
	}

	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		if id, ok := b.dict[string(p)]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func splitOnRune(b []byte, r rune) [][]byte {
	var out [][]byte
	last := 0
	for i := 0; i < len(b); {
		ch, size := utf8.DecodeRune(b[i:])
		if ch == r {
			out = append(out, b[last:i])
			last = i + size
		}
		i += size
	}
	out = append(out, b[last:])
	return out
}

func endsWithSpace(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	ch, _ := utf8.DecodeLastRune(b)
	return ch == ' '
}