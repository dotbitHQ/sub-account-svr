package rule

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRuleSpecialCharacters(t *testing.T) {
	rule := NewSubAccountRuleSlice()

	price := 100000000

	err := rule.Parser([]byte(fmt.Sprintf(`
[
    {
        "name": "特殊字符账户",
        "note": "",
        "price": %d,
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
`, price)))
	if err != nil {
		t.Fatal(err)
	}

	hit, _, err := rule.Hit("jerry.bit")
	assert.NoError(t, err)
	assert.False(t, hit)

	hit, idx, err := rule.Hit("jerry✅.bit")
	assert.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, idx, 0)
	assert.EqualValues(t, (*rule)[idx].Price, price)

	hit, _, err = rule.Hit("jerry💚.bit")
	assert.NoError(t, err)
	assert.False(t, hit)
}

func TestAccountLengthPrice(t *testing.T) {
	rule := NewSubAccountRuleSlice()

	price := 100000000

	err := rule.Parser([]byte(fmt.Sprintf(`
[
    {
        "name": "特殊字符账户",
        "note": "",
        "price": %d,
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
`, price)))
	if err != nil {
		t.Fatal(err)
	}
}
