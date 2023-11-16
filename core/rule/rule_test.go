//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package rule

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	trpc "trpc.group/trpc-go/trpc-go"
)

func fakeGetIntStr(context.Context, string) string {
	return "666"
}

func fakeGetStr(context.Context, string) string {
	return "abc"
}

func TestMatchRule(t *testing.T) {
	ctx := trpc.BackgroundContext()

	// empty rule
	matched, err := MatchRule(ctx, nil, fakeGetIntStr)
	assert.Nil(t, err)
	assert.False(t, matched)

	// empty expression
	ruleItem := &entity.RuleItem{
		Expression: "",
	}
	matched, err = MatchRule(ctx, ruleItem, fakeGetIntStr)
	assert.Nil(t, err)
	assert.False(t, matched)

	// init rule failed
	ruleItem.Conditions = []*entity.Condition{
		{Key: "a", Val: "5", Oper: ">"},
		{Key: "b", Val: "5", Oper: ">="},
		{Key: "c", Val: "5", Oper: "<"},
		{Key: "d", Val: "\\uFFFD", Oper: RegexpOpt},
	}
	ruleItem.Expression = "0&&1||"
	assert.NotNil(t, FormatRule(ruleItem))

	// match success
	ruleItem.Conditions = []*entity.Condition{
		{Key: "a", Val: "5", Oper: ">"},
		{Key: "b", Val: "5", Oper: ">="},
		{Key: "c", Val: "5", Oper: "<"},
		{Key: "d", Val: "b", Oper: "<="},
	}
	ruleItem.Expression = "0&&1||"
	assert.Nil(t, FormatRule(ruleItem))
	matched, err = MatchRule(ctx, ruleItem, fakeGetIntStr)
	assert.Nil(t, err)
	assert.True(t, matched)

	ruleItem.Expression = "0&&2"
	assert.Nil(t, FormatRule(ruleItem))
	matched, err = MatchRule(ctx, ruleItem, fakeGetIntStr)
	assert.Nil(t, err)
	assert.False(t, matched)

	ruleItem.Expression = "0||2"
	assert.Nil(t, FormatRule(ruleItem))
	matched, err = MatchRule(ctx, ruleItem, fakeGetIntStr)
	assert.Nil(t, err)
	assert.True(t, matched)

	ruleItem.ConditionIdxList = []int{5, 6}
	ruleItem.OptList = []string{"||"}
	matched, err = MatchRule(ctx, ruleItem, fakeGetIntStr)
	assert.NotNil(t, err)
	assert.False(t, matched)
}

func TestParseRuleExpression(t *testing.T) {
	exp := "0&&1||2||3"
	idxList, optList, err := ParseRuleExpression(exp, 4)
	assert.Nil(t, err)
	assert.Equal(t, idxList[2], 2)
	assert.Equal(t, optList[0], "&&")
	exp = "0&&1||2||"
	idxList, optList, err = ParseRuleExpression(exp, 3)
	assert.Nil(t, err)
	assert.ElementsMatch(t, idxList, []int{0, 1, 2})
	assert.ElementsMatch(t, optList, []string{"&&", "||"})

	exp = "0&&||2||"
	_, _, err = ParseRuleExpression(exp, 2)
	assert.NotNil(t, err)

	exp = "||0&&1||2||"
	idxList, optList, err = ParseRuleExpression(exp, 3)
	assert.Nil(t, err)
	assert.ElementsMatch(t, idxList, []int{0, 1, 2})
	assert.ElementsMatch(t, optList, []string{"&&", "||"})

	exp = "0&&1&"
	idxList, optList, err = ParseRuleExpression(exp, 2)
	assert.Nil(t, err)
	assert.ElementsMatch(t, idxList, []int{0, 1})
	assert.ElementsMatch(t, optList, []string{"&&"})

	exp = "&&0"
	idxList, optList, err = ParseRuleExpression(exp, 1)
	assert.Nil(t, err)
	assert.ElementsMatch(t, idxList, []int{0})
	assert.ElementsMatch(t, optList, []string{})

	exp = "0&&-1"
	_, _, err = ParseRuleExpression(exp, 3)
	assert.NotNil(t, err)

	exp = "1&&0"
	_, _, err = ParseRuleExpression(exp, 3)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not sorted")

	exp = ""
	_, _, err = ParseRuleExpression(exp, 3)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "empty expression or condition")

	exp = "0&&1*"
	_, _, err = ParseRuleExpression(exp, 1)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid max idx")

	exp = "0"
	_, _, err = ParseRuleExpression(exp, 1)
	assert.Nil(t, err)
}

func TestFormatRule(t *testing.T) {

	err := FormatRule(nil)
	assert.NotNil(t, err)

	ruleItem := &entity.RuleItem{
		Expression: "0&&1||2",
		Conditions: []*entity.Condition{
			{},
			{},
			{},
		},
	}
	err = FormatRule(ruleItem)
	assert.Nil(t, err)

	ruleItem = &entity.RuleItem{
		Expression: "0&&1||-2",
		Conditions: []*entity.Condition{
			{},
			{},
			{},
		},
	}
	err = FormatRule(ruleItem)
	assert.NotNil(t, err)
}

func TestJudgeCondition(t *testing.T) {
	cond := &entity.Condition{
		Key:  "a",
		Val:  "5",
		Oper: "invalid",
	}
	ok := judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.False(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "5",
		Oper: ">",
	}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "b",
		Oper: "<",
	}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)
	cond = &entity.Condition{
		Key:  "a",
		Val:  "b",
		Oper: "<",
	}
	ok = judgeCondition(context.Background(), cond, fakeGetStr)
	assert.True(t, ok)
	cond = &entity.Condition{Key: "a", Val: "666", Oper: "=="}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)
	cond = &entity.Condition{Key: "a", Val: "6666", Oper: "!="}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{Key: "a", Val: "666", Oper: "<="}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{Key: "a", Val: "666", Oper: ">="}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{Key: "a", Val: "666,111", Oper: "in"}
	err := parseRuleConditionVal([]*entity.Condition{cond})
	assert.Nil(t, err)
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{Key: "a", Val: "6666,111", Oper: "!in"}
	err = parseRuleConditionVal([]*entity.Condition{cond})
	assert.Nil(t, err)
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "666,111",
		Oper: "!in",
	}
	err = parseRuleConditionVal([]*entity.Condition{cond})
	assert.Nil(t, err)
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.False(t, ok)

	// 测试 val 携带空格能否清理掉
	cond = &entity.Condition{
		Key:  "a",
		Val:  "666 ,111, 222,555",
		Oper: "in",
	}
	err = parseRuleConditionVal([]*entity.Condition{cond})
	assert.Nil(t, err)
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "6666, 555, 222,111",
		Oper: "!in",
	}
	err = parseRuleConditionVal([]*entity.Condition{cond})
	assert.Nil(t, err)
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "666 ,111, 222 ",
		Oper: "!in",
	}
	err = parseRuleConditionVal([]*entity.Condition{cond})
	assert.Nil(t, err)
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.False(t, ok)

	// 测试携带非预期 parsedVal
	cond = &entity.Condition{
		Key:  "a",
		Val:  "666,111",
		Oper: "in",
	}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.False(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "6666,111",
		Oper: "!in",
	}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.False(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "666,111",
		Oper: "!in",
	}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.False(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "^666$",
		Oper: RegexpOpt,
	}
	err = parseRuleConditionVal([]*entity.Condition{cond})
	assert.Nil(t, err)
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.True(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "^666$",
		Oper: RegexpOpt,
	}
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.False(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "^6667$",
		Oper: RegexpOpt,
	}
	err = parseRuleConditionVal([]*entity.Condition{cond})
	assert.Nil(t, err)
	ok = judgeCondition(context.Background(), cond, fakeGetIntStr)
	assert.False(t, ok)

	cond = &entity.Condition{
		Key:  "a",
		Val:  "\\uFFFD",
		Oper: RegexpOpt,
	}
	err = parseRuleConditionVal([]*entity.Condition{cond})
	assert.NotNil(t, err)
}

func TestReg(t *testing.T) {
	idList := idxGetReg.FindAllString("0&&1||2", -1)
	t.Log(idList)
}
