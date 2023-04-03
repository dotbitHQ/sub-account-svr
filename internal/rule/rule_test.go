package rule

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAccountId(t *testing.T) {
	accounts := []string{"test.bit", "reverse.bit"}
	outs := make([]string, 0)
	for _, v := range accounts {
		out := common.Bytes2Hex(common.Blake2b([]byte(v))[:20])
		outs = append(outs, out)
	}
	t.Log(outs)
}

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

func TestRuleWhitelist(t *testing.T) {
	rule := NewSubAccountRuleSlice()

	price := 100000000

	err := rule.Parser([]byte(fmt.Sprintf(`
[
    {
        "name": "特殊账户",
        "note": "",
        "price": %d,
        "ast": {
            "type": "function",
            "name": "in_list",
            "arguments": [
                {
                    "type": "variable",
                    "name": "account"
                },
                {
                    "type": "value",
                    "value_type": "binary[]",
                    "value": [
                        "0xb28072bd0201e6feeb4cd96a6879d6422f2218cd",
                        "0x75bc2d3192ec310b6ac2f826d3e19a5cfe9f080a"
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

	hit, _, err = rule.Hit("test.bit")
	assert.NoError(t, err)
	assert.True(t, hit)

	hit, _, err = rule.Hit("reverse.bit")
	assert.NoError(t, err)
	assert.True(t, hit)
}
