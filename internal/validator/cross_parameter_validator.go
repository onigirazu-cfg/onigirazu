package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CrossParameterValidator handles cross-parameter validation
type CrossParameterValidator struct {
	rules  []types.CrossParameterRule
	params map[string]interface{}
}

// NewCrossParameterValidator creates a new cross-parameter validator
func NewCrossParameterValidator(rules []types.CrossParameterRule) *CrossParameterValidator {
	return &CrossParameterValidator{
		rules: rules,
	}
}

// ValidateCrossParameters validates parameters against cross-parameter rules
func (cpv *CrossParameterValidator) ValidateCrossParameters(params map[string]interface{}) *types.ValidationResult {
	result := &types.ValidationResult{
		Valid:            true,
		Errors:           []types.ParameterValidationError{},
		CrossParamErrors: []types.CrossParameterValidationError{},
	}

	if len(cpv.rules) == 0 {
		return result
	}

	cpv.params = params

	for _, rule := range cpv.rules {
		if ok, err := cpv.evaluateRule(rule); !ok {
			result.Valid = false
			result.CrossParamErrors = append(result.CrossParamErrors, err)
		}
	}

	return result
}

// evaluateRule evaluates a single cross-parameter rule
func (cpv *CrossParameterValidator) evaluateRule(rule types.CrossParameterRule) (bool, types.CrossParameterValidationError) {
	ok, details := cpv.evaluateExpression(rule.Rule)

	if !ok {
		return false, types.CrossParameterValidationError{
			Rule:     rule.Rule,
			Error:    fmt.Sprintf("cross-parameter rule validation failed: %s", rule.Rule),
			ErrorMsg: rule.ErrorMsg,
			Details:  details,
		}
	}

	return true, types.CrossParameterValidationError{}
}

// evaluateExpression evaluates a rule expression and returns the result
// Supports: param=value, param!=value, param>value, param<value, param>=value, param<=value
// And logical operators: && (AND), || (OR)
// Example: "port=80 && service=http" or "enable_auth=true || admin_override=true"
func (cpv *CrossParameterValidator) evaluateExpression(expr string) (bool, map[string]interface{}) {
	details := make(map[string]interface{})
	expr = strings.TrimSpace(expr)

	// Handle OR expressions (lower precedence)
	if parts := cpv.splitByOperator(expr, "||"); len(parts) > 1 {
		for _, part := range parts {
			if ok, _ := cpv.evaluateExpression(strings.TrimSpace(part)); ok {
				return true, details
			}
		}
		return false, details
	}

	// Handle AND expressions (higher precedence)
	if parts := cpv.splitByOperator(expr, "&&"); len(parts) > 1 {
		for _, part := range parts {
			if ok, partDetails := cpv.evaluateExpression(strings.TrimSpace(part)); !ok {
				for k, v := range partDetails {
					details[k] = v
				}
				return false, details
			}
		}
		return true, details
	}

	// Evaluate single comparison
	return cpv.evaluateComparison(expr, details)
}

// splitByOperator splits expression by operator, respecting nested parentheses
func (cpv *CrossParameterValidator) splitByOperator(expr, op string) []string {
	var parts []string
	var current strings.Builder
	depth := 0

	for i := 0; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
			current.WriteByte(expr[i])
		} else if expr[i] == ')' {
			depth--
			current.WriteByte(expr[i])
		} else if depth == 0 && i+len(op) <= len(expr) && expr[i:i+len(op)] == op {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			i += len(op) - 1
		} else {
			current.WriteByte(expr[i])
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	if len(parts) <= 1 {
		return []string{expr}
	}

	return parts
}

// evaluateComparison evaluates a single comparison (e.g., "port=80")
func (cpv *CrossParameterValidator) evaluateComparison(comparison string, details map[string]interface{}) (bool, map[string]interface{}) {
	comparison = strings.TrimSpace(comparison)

	// Try each operator
	operators := []string{">=", "<=", "!=", "=", ">", "<"}
	for _, op := range operators {
		if idx := strings.Index(comparison, op); idx != -1 {
			return cpv.evaluateOperator(comparison, op, idx, details)
		}
	}

	return false, details
}

// evaluateOperator evaluates a comparison with a specific operator
func (cpv *CrossParameterValidator) evaluateOperator(comparison, op string, idx int, details map[string]interface{}) (bool, map[string]interface{}) {
	paramName := strings.TrimSpace(comparison[:idx])
	expectedStr := strings.TrimSpace(comparison[idx+len(op):])

	paramValue, exists := cpv.params[paramName]
	if !exists {
		details[paramName] = "parameter not provided"
		return false, details
	}

	details[paramName] = paramValue
	details[fmt.Sprintf("%s_expected", paramName)] = expectedStr

	switch op {
	case "=":
		return cpv.compareEqual(paramValue, expectedStr), details
	case "!=":
		return !cpv.compareEqual(paramValue, expectedStr), details
	case ">":
		return cpv.compareGreater(paramValue, expectedStr), details
	case "<":
		return cpv.compareLess(paramValue, expectedStr), details
	case ">=":
		return cpv.compareGreaterEqual(paramValue, expectedStr), details
	case "<=":
		return cpv.compareLessEqual(paramValue, expectedStr), details
	}

	return false, details
}

// compareEqual compares if parameter value equals expected value
func (cpv *CrossParameterValidator) compareEqual(value interface{}, expected string) bool {
	switch v := value.(type) {
	case string:
		return v == expected
	case bool:
		return strconv.FormatBool(v) == strings.ToLower(expected) ||
			v == (expected == "true" || expected == "1" || expected == "yes")
	case int:
		expInt, err := strconv.Atoi(expected)
		return err == nil && v == expInt
	case float64:
		// For float, check if it's a whole number
		if v == float64(int64(v)) {
			expInt, err := strconv.Atoi(expected)
			return err == nil && int64(v) == int64(expInt)
		}
		expFloat, err := strconv.ParseFloat(expected, 64)
		return err == nil && v == expFloat
	case int64:
		expInt, err := strconv.ParseInt(expected, 10, 64)
		return err == nil && v == expInt
	}

	return fmt.Sprintf("%v", value) == expected
}

// compareGreater compares if parameter value is greater than expected
func (cpv *CrossParameterValidator) compareGreater(value interface{}, expected string) bool {
	// Try numeric comparison first
	cmp := cpv.numericCompare(value, expected)
	if cmp > 0 {
		return true
	}

	// Fall back to string comparison
	return fmt.Sprintf("%v", value) > expected
}

// compareLess compares if parameter value is less than expected
func (cpv *CrossParameterValidator) compareLess(value interface{}, expected string) bool {
	// Try numeric comparison first
	cmp := cpv.numericCompare(value, expected)
	if cmp < 0 {
		return true
	}

	// Fall back to string comparison
	return fmt.Sprintf("%v", value) < expected
}

// compareGreaterEqual compares if parameter value is >= expected
func (cpv *CrossParameterValidator) compareGreaterEqual(value interface{}, expected string) bool {
	cmp := cpv.numericCompare(value, expected)
	if cmp >= 0 {
		return true
	}

	return fmt.Sprintf("%v", value) >= expected
}

// compareLessEqual compares if parameter value is <= expected
func (cpv *CrossParameterValidator) compareLessEqual(value interface{}, expected string) bool {
	cmp := cpv.numericCompare(value, expected)
	if cmp <= 0 {
		return true
	}

	return fmt.Sprintf("%v", value) <= expected
}

// numericCompare performs numeric comparison
// Returns: positive if value > expected, negative if value < expected, 0 if equal
// Returns math.MaxInt64 if comparison fails
func (cpv *CrossParameterValidator) numericCompare(value interface{}, expected string) int64 {
	// Convert value to float64
	var floatVal float64
	switch v := value.(type) {
	case int:
		floatVal = float64(v)
	case int64:
		floatVal = float64(v)
	case float64:
		floatVal = v
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 9223372036854775807 // MaxInt64 to signal failure
		}
		floatVal = f
	default:
		return 9223372036854775807
	}

	// Convert expected to float64
	expectedFloat, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return 9223372036854775807
	}

	if floatVal > expectedFloat {
		return 1
	} else if floatVal < expectedFloat {
		return -1
	}
	return 0
}

// ParseRules parses rule strings from YAML format
func ParseRules(rulesData []interface{}) []types.CrossParameterRule {
	var rules []types.CrossParameterRule

	for _, ruleData := range rulesData {
		switch r := ruleData.(type) {
		case types.CrossParameterRule:
			rules = append(rules, r)
		case map[string]interface{}:
			rule := types.CrossParameterRule{
				Rule:        fmt.Sprintf("%v", r["rule"]),
				Description: fmt.Sprintf("%v", r["description"]),
				ErrorMsg:    fmt.Sprintf("%v", r["error"]),
				Severity:    fmt.Sprintf("%v", r["severity"]),
			}
			if rule.Severity == "" || rule.Severity == "<nil>" {
				rule.Severity = "error"
			}
			rules = append(rules, rule)
		}
	}

	return rules
}

// ValidateRuleExpression validates that a rule expression is well-formed
func ValidateRuleExpression(expr string) error {
	expr = strings.TrimSpace(expr)

	if expr == "" {
		return fmt.Errorf("rule expression cannot be empty")
	}

	// Check for matching parentheses
	depth := 0
	for _, ch := range expr {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth < 0 {
				return fmt.Errorf("unmatched closing parenthesis")
			}
		}
	}
	if depth != 0 {
		return fmt.Errorf("unmatched opening parenthesis")
	}

	// Check that each comparison has a valid operator
	// Simple check: must contain at least one of: =, !=, >, <, >=, <=
	hasOperator := false
	operators := []string{">=", "<=", "!=", "=", ">", "<"}
	for _, op := range operators {
		if strings.Contains(expr, op) {
			hasOperator = true
			break
		}
	}

	if !hasOperator {
		return fmt.Errorf("rule expression must contain at least one comparison operator (=, !=, >, <, >=, <=)")
	}

	// Check that parameter names are valid (alphanumeric and underscore)
	paramRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	params := paramRegex.FindAllString(expr, -1)
	if len(params) == 0 {
		return fmt.Errorf("rule expression must contain at least one parameter name")
	}

	// Check logical operators are used correctly
	if strings.Contains(expr, "&&") || strings.Contains(expr, "||") {
		// Make sure they're surrounded by whitespace or are at boundaries
		logicRegex := regexp.MustCompile(`(\s(&&|\|\|)\s)|^(&&|\|\|)|.*(&&|\|\|)$`)
		if !logicRegex.MatchString(expr) {
			// More lenient check
			if !strings.Contains(expr, " && ") && !strings.Contains(expr, " || ") &&
				!strings.Contains(expr, "&&") && !strings.Contains(expr, "||") {
				return fmt.Errorf("logical operators must be properly formatted")
			}
		}
	}

	return nil
}
