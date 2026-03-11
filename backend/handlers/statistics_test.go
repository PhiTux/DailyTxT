package handlers

import (
	"strings"
	"testing"
)

func TestCountWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		// --- 基础英文与数字 ---
		{
			name:     "Simple English",
			input:    "Hello world",
			expected: 2,
		},
		{
			name:     "Numbers only",
			input:    "123 456",
			expected: 2,
		},
		{
			name:     "Alphanumeric",
			input:    "User123 logged in",
			expected: 3, // User123, logged, in
		},
		{
			name:     "Empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "Only punctuation",
			input:    "... !!! ???",
			expected: 0,
		},
		{
			name:     "Only whitespace",
			input:    "   \t\n  ",
			expected: 0,
		},

		// --- 连字符与撇号逻辑 ---
		{
			name:     "Hyphenated words",
			input:    "well-known test-case state-of-the-art",
			expected: 3,
		},
		{
			name:     "Contractions with apostrophe",
			input:    "don't can't O'Neill it's",
			expected: 4,
		},
		{
			name:     "Trailing hyphen (should split)",
			input:    "test- case",
			expected: 2,
		},
		{
			name:     "Leading hyphen (should split)",
			input:    "test -case",
			expected: 2,
		},
		{
			name:     "Hyphen between non-words",
			input:    "123 - 456",
			expected: 2,
		},
		{
			name:     "Multiple apostrophes",
			input:    "rock'n'roll",
			expected: 1,
		},

		// --- 中文 ---
		{
			name:     "Simple Chinese",
			input:    "你好世界",
			expected: 4,
		},
		{
			name:     "Chinese with punctuation",
			input:    "你好，世界！",
			expected: 4,
		},
		{
			name:     "Chinese Extension A (罕见字)",
			input:    "\u3400\u3401", // 㐀㐁
			expected: 2,
		},
		{
			name:     "Chinese Compatibility (兼容汉字)",
			input:    "\uFA00\uFA01", // 豈更
			expected: 2,
		},

		// --- 日文 (假名) ---
		{
			name:     "Hiragana",
			input:    "こんにちは",
			expected: 5,
		},
		{
			name:     "Katakana",
			input:    "コンニチワ",
			expected: 5,
		},
		{
			name:     "Mixed Kana",
			input:    "アイウエオ あいうえお",
			expected: 10,
		},
		{
			name:     "Kana with punctuation",
			input:    "こんにちは。",
			expected: 5,
		},

		// --- 韩文 ---
		{
			name:     "Simple Hangul",
			input:    "안녕하세요",
			expected: 5,
		},
		{
			name:     "Hangul with space",
			input:    "안녕 세계",
			expected: 4, // 안(1) 녕(1) 세(1) 계(1)
		},

		// --- 多语言混合 ---
		{
			name:     "English + Chinese",
			input:    "Hello 世界",
			expected: 3, // Hello(1) + 世(1) + 界(1)
		},
		{
			name:  "Complex Mixed Sentence",
			input: "Hello 世界，这是一个 test-case 示例！こんにちは",
			// Hello(1) + 世(1) + 界(1) + 这(1) + 是(1) + 一(1) + 个(1) + test-case(1)
			//  + 示(1) + 例(1) + こ(1) + ん(1) + に(1) + ち(1) + は(1)
			expected: 15,
		},
		{
			name:     "English + Korean + Numbers",
			input:    "Test123 안녕 456",
			expected: 4, // Test123(1) + 안(1) + 녕(1) + 456(1)
		},

		// --- 特殊字符与变音符号 ---
		{
			name:     "Latin with diacritics",
			input:    "café résumé naïve",
			expected: 3,
		},
		{
			name:     "Emoji (should be ignored)",
			input:    "Hello 🌍 World 👋",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountWords(tt.input)
			if result != tt.expected {
				t.Errorf("CountWords(%q) = %d; expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

// Some extreme edge cases
func TestCountWords_EdgeCases(t *testing.T) {
	// Test with a very large string.
	var largeText strings.Builder
	for range 100000 {
		largeText.WriteString("word ")
	}
	if got := CountWords(largeText.String()); got != 100000 {
		t.Errorf("Large text test failed: got %d, want 100000", got)
	}

	// Test trailing hyphen at the end of the string.
	if got := CountWords("test-"); got != 1 {
		t.Errorf("Trailing hyphen test failed: got %d, want 1", got)
	}

	// Test leading hyphen at the start of the string.
	if got := CountWords("-test"); got != 1 {
		t.Errorf("Leading hyphen test failed: got %d, want 1", got)
	}

	// Test consecutive hyphens (should not form a single word).
	if got := CountWords("a--b"); got != 2 {
		t.Errorf("Double hyphen test failed: got %d, want 2", got)
	}
}
