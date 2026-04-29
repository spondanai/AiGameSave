package adapters

import (
	"testing"

	"github.com/spondanai/aigamesave/internal/domain"
)

func TestSelectContext_AnchorAtSubstantialUser(t *testing.T) {
	turns := []domain.Turn{
		{Role: "user", Content: "เพิ่ม adapter ของ vscode ให้หน่อย"},
		{Role: "assistant", Content: "กำลังสร้าง..."},
		{Role: "user", Content: "โอเค"},
		{Role: "assistant", Content: "สร้างเสร็จแล้ว"},
		{Role: "assistant", Content: "รันเทสต์ผ่าน"},
		{Role: "assistant", Content: "push ขึ้น main แล้ว"},
	}
	got := selectContext(turns)

	// anchor = turn[0] (last substantial user), tail = last 5 of turns[1:]
	if got[0].Role != "user" || got[0].Content != "เพิ่ม adapter ของ vscode ให้หน่อย" {
		t.Errorf("expected anchor to be the substantial user turn, got %+v", got[0])
	}
	if len(got) != 6 {
		t.Errorf("expected 6 turns (anchor + 5 tail), got %d", len(got))
	}
}

func TestSelectContext_SkipsShortAcknowledgements(t *testing.T) {
	turns := []domain.Turn{
		{Role: "user", Content: "ช่วยเพิ่มฟีเจอร์ auto-detect ด้วย"},
		{Role: "assistant", Content: "เพิ่มแล้ว"},
		{Role: "user", Content: "โอเค"},       // trivial — 4 runes
		{Role: "user", Content: "ลองรันดู"},   // trivial — 8 runes
		{Role: "assistant", Content: "ผ่าน"},
	}
	got := selectContext(turns)

	// anchor should be the first user turn, not the trivial ones
	if got[0].Content != "ช่วยเพิ่มฟีเจอร์ auto-detect ด้วย" {
		t.Errorf("expected anchor to be the first substantial user turn, got %+v", got[0])
	}
}

func TestSelectContext_TailCappedAtMaxTailTurns(t *testing.T) {
	turns := []domain.Turn{
		{Role: "user", Content: "รีแฟคเตอร์โค้ดทั้งหมดให้สะอาดขึ้นหน่อย"},
	}
	// Simulate 10 agentic assistant turns after the anchor
	for range 10 {
		turns = append(turns, domain.Turn{Role: "assistant", Content: "ทำอยู่..."})
	}

	got := selectContext(turns)
	// anchor + 5 tail (the last 5 of 10)
	if len(got) != maxTailTurns+1 {
		t.Errorf("expected %d turns, got %d", maxTailTurns+1, len(got))
	}
	if got[0].Content != "รีแฟคเตอร์โค้ดทั้งหมดให้สะอาดขึ้นหน่อย" {
		t.Errorf("anchor not preserved: %+v", got[0])
	}
}

func TestSelectContext_FallbackWhenNoSubstantialUser(t *testing.T) {
	turns := make([]domain.Turn, 10)
	for i := range turns {
		turns[i] = domain.Turn{Role: "assistant", Content: "ข้อความ"}
	}
	got := selectContext(turns)
	if len(got) != fallbackTurns {
		t.Errorf("expected fallback of %d turns, got %d", fallbackTurns, len(got))
	}
}

func TestIsSubstantial(t *testing.T) {
	cases := []struct {
		content string
		want    bool
	}{
		{"โอเค", false},           // 4 runes
		{"ครับ", false},           // 4 runes
		{"ลองรันดู", false},       // 8 runes
		{"ok", false},             // 2 runes
		{"yes", false},            // 3 runes
		{"เพิ่ม adapter ของ vscode ให้หน่อย", true}, // 32 runes
		{"ช่วยเพิ่มฟีเจอร์ด้วย", true},             // 21 runes
		{"1234567890", true},      // exactly 10 runes — at threshold
		{"123456789", false},      // 9 runes — below threshold
	}
	for _, c := range cases {
		if got := isSubstantial(c.content); got != c.want {
			t.Errorf("isSubstantial(%q) = %v, want %v", c.content, got, c.want)
		}
	}
}
