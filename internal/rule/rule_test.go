package rule

import (
	"testing"
)

func TestRule(t *testing.T) {
	rule := NewSubAccountRuleSlice()
	err := rule.Parser([]byte(`
[
    {
        "name": "特殊字符账户",
        "note": "",
        "price": 100000000,
        "ast": {
            "type": "function",
            "name": "include_chars",
            "arguments": [
                {
                    "type": "variable",
                    "name": "account_chars"
                },
                {
                    "type": "value",
                    "value_type": "string[]",
                    "value": [
                        "⚠️",
                        "❌",
                        "✅"
                    ]
                }
            ]
        }
    }
]
`))

	if err != nil {
		t.Fatal(err)
	}
}
