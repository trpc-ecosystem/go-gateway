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

// Package rule encapsulates a simple rule matching mechanism for dynamic parameter matching
package rule

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
)

// parsedB is the parsed object of b during router initialization, used for matching during requests
type compareFunc func(a, b string, parsedB interface{}) bool

const (
	// AndOpt represents the logical AND operator
	AndOpt = "&&"
	// OrOpt represents the logical OR operator
	OrOpt = "||"

	// GreaterThanOpt represents the greater than operator
	GreaterThanOpt = ">"
	// GreaterOrEqualToOpt represents the greater than or equal to operator
	GreaterOrEqualToOpt = ">="
	// LessThanOpt represents the less than operator
	LessThanOpt = "<"
	// LessThanOrEqualToOpt represents the less than or equal to operator
	LessThanOrEqualToOpt = "<="
	// EqualToOpt represents the equal to operator
	EqualToOpt = "=="
	// NotEqualToOpt represents the not equal to operator
	NotEqualToOpt = "!="
	// InOpt represents the in operator
	InOpt = "in"
	// NotInOpt represents the not in operator
	NotInOpt = "!in"
	// RegexpOpt represents the regular expression matching operator
	RegexpOpt = "regexp"
)

var (
	// Regular expression to extract condition index
	idxGetReg = regexp.MustCompile(`\d+`)

	compareFuncs = map[string]compareFunc{
		GreaterThanOpt:       gt,
		GreaterOrEqualToOpt:  ge,
		LessThanOpt:          lt,
		LessThanOrEqualToOpt: le,
		EqualToOpt:           func(a, b string, _ interface{}) bool { return a == b },
		NotEqualToOpt:        func(a, b string, _ interface{}) bool { return a != b },
		InOpt:                in,
		NotInOpt:             notIn,
		RegexpOpt:            regexpMatch,
	}
)

// GetStringFunc defines the function to retrieve parameters
// For HTTP protocol, it can retrieve form, query, header, and cookie parameters; for RPC, it can retrieve metadata
type GetStringFunc func(ctx context.Context, key string) string

// MatchRule performs rule matching
func MatchRule(ctx context.Context, ruleItem *entity.RuleItem, getString GetStringFunc) (bool, error) {
	if ruleItem == nil {
		return false, nil
	}
	idxList, optList := ruleItem.ConditionIdxList, ruleItem.OptList
	if len(idxList) == 0 {
		return false, nil
	}
	cNum := len(ruleItem.Conditions)
	position := idxList[0]
	if idxList[0] >= cNum {
		return false, errs.New(gerrs.ErrWrongConfig, "error conditions conf")
	}

	flag := judgeCondition(ctx, ruleItem.Conditions[position], getString)

	for i, op := range optList {
		position = idxList[i+1]
		if position >= cNum {
			continue
		}
		rFlag := judgeCondition(ctx, ruleItem.Conditions[position], getString)

		switch op {
		case AndOpt:
			flag = flag && rFlag
		case OrOpt:
			flag = flag || rFlag
		default:
			return false, errs.Newf(gerrs.ErrWrongConfig, "invalid option:%s", op)
		}
	}
	return flag, nil
}

// FormatRule formats the rule
func FormatRule(ruleItem *entity.RuleItem) error {
	if ruleItem == nil || ruleItem.Expression == "" {
		return errs.New(gerrs.ErrWrongConfig, "empty rule item")
	}
	idxList, optList, err := ParseRuleExpression(ruleItem.Expression, len(ruleItem.Conditions))
	if err != nil {
		return gerrs.Wrap(err, "parse rule expression err")
	}

	ruleItem.ConditionIdxList = idxList
	ruleItem.OptList = optList
	if err := parseRuleConditionVal(ruleItem.Conditions); err != nil {
		return gerrs.Wrap(err, "parse rule condition val err")
	}
	return nil
}

// parseRuleConditionVal parses the condition matching values
func parseRuleConditionVal(conditions []*entity.Condition) error {
	// Iterate over conditions
	for _, cond := range conditions {
		switch cond.Oper {
		case InOpt, NotInOpt:
			condValList := strings.Split(cond.Val, ",")
			// Trim spaces
			for i := range condValList {
				condValList[i] = strings.TrimSpace(condValList[i])
			}
			cond.ParsedVal = convert.StrSlice2Map(condValList)
		case RegexpOpt:
			exp, err := regexp.Compile(cond.Val)
			if err != nil {
				return errs.Wrap(err, gerrs.ErrWrongConfig, "compile rule regexp err")
			}
			cond.ParsedVal = exp
		}
	}
	return nil
}

// ParseRuleExpression parses the condition expression
// Example input: "0&&1||2"
func ParseRuleExpression(expr string, condCount int) ([]int, []string, error) {
	if expr == "" || condCount == 0 {
		return nil, nil, errs.New(gerrs.ErrWrongConfig, "empty expression or condition")
	}
	// Get the index list of the condition expression, e.g., for expression "0&&1||2", get [0 1 2]
	idxStrList := idxGetReg.FindAllString(expr, -1)
	// Convert to int slice
	idxList, err := convert.ToIntSlice(idxStrList)
	if err != nil {
		return nil, nil, gerrs.Wrap(err, "convert rule idx err")
	}
	// Get the logical operators, e.g., for expression "0&&1||2", get ["&&", "||"]
	optList := idxGetReg.Split(expr, -1)
	// Remove leading and trailing empty strings from the result of "0&&1" split, resulting in ["", "&&", ""]
	if len(optList) >= 2 {
		optList = optList[1 : len(optList)-1]
	}

	// Validate the operators
	for _, o := range optList {
		if o != AndOpt && o != OrOpt {
			return nil, nil, errs.Newf(gerrs.ErrWrongConfig, "invalid opt:%s", o)
		}
	}
	// Validate the index positions: the maximum index should be less than the number of conditions, and the minimum
	// index should be greater than 0 (already checked by the regular expression)

	tmpSlice := sort.IntSlice(idxList)
	if !sort.IsSorted(sort.IntSlice(idxList)) {
		return nil, nil, errs.New(gerrs.ErrWrongConfig, "rule idx is not sorted")
	}

	maxIdx := tmpSlice[len(tmpSlice)-1]
	if maxIdx+1 > condCount {
		return nil, nil, errs.Newf(gerrs.ErrWrongConfig,
			"invalid max idx:%v,cond count:%v", maxIdx, condCount)
	}

	return idxList, optList, nil
}

// judgeCondition filters a single condition, currently only supports ">,>=,<,<=,==,!=,in,!in,regexp" operators
func judgeCondition(ctx context.Context, cond *entity.Condition, getString GetStringFunc) bool {
	compare, ok := compareFuncs[cond.Oper]
	if !ok {
		// Unsupported operator
		return false
	}
	val := getString(ctx, cond.Key)
	return compare(val, cond.Val, cond.ParsedVal)
}

// getNumFromString converts numbers
func getNumFromString(a, b string) (int64, int64, error) {
	aParsed, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		return 0, 0, err
	}

	bParsed, err := strconv.ParseInt(b, 10, 64)
	return aParsed, bParsed, err
}

// gt is greater than
func gt(a, b string, _ interface{}) bool {
	// If both are numbers, compare them as numbers; otherwise, compare them as strings
	if s, d, err := getNumFromString(a, b); err == nil {
		return s > d
	}

	return a > b
}

// ge is greater than or equal to
func ge(a, b string, _ interface{}) bool {
	// If both are numbers, compare them as numbers; otherwise, compare them as strings
	if s, d, err := getNumFromString(a, b); err == nil {
		return s >= d
	}

	return a >= b
}

// lt is less than
func lt(a, b string, _ interface{}) bool {
	// If both are numbers, compare them as numbers; otherwise, compare them as strings
	if s, d, err := getNumFromString(a, b); err == nil {
		return s < d
	}

	return a < b
}

// le is less than or equal to
func le(a, b string, _ interface{}) bool {
	// If both are numbers, compare them as numbers; otherwise, compare them as strings
	if s, d, err := getNumFromString(a, b); err == nil {
		return s <= d
	}

	return a <= b
}

// in checks if a value is in the list
func in(a, _ string, parsedB interface{}) bool {
	valMap, ok := parsedB.(map[string]struct{})
	if !ok {
		log.Errorf("invalid parsedB type:%T", parsedB)
		return false
	}
	_, ok = valMap[a]
	return ok
}

// notIn checks if a value is not in the list
func notIn(a, _ string, parsedB interface{}) bool {
	valMap, ok := parsedB.(map[string]struct{})
	if !ok {
		log.Errorf("invalid parsedB type:%T", parsedB)
		return false
	}
	_, ok = valMap[a]
	return !ok
}

// regexpMatch performs regular expression matching
func regexpMatch(a, _ string, parsedB interface{}) bool {
	r, ok := parsedB.(*regexp.Regexp)
	if !ok {
		log.Errorf("invalid parsedB type:%T", parsedB)
		return false
	}
	return r.MatchString(a)
}
