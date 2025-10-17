package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ConditionEvaluator evaluates conditional requirements for parameters
type ConditionEvaluator struct {
	parameters map[string]interface{}
}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator(parameters map[string]interface{}) *ConditionEvaluator {
	return &ConditionEvaluator{
		parameters: parameters,
	}
}

// EvaluateCondition evaluates a condition expression
// Supported formats:
//   - parameter=value
//   - parameter!=value
//   - parameter>value
//   - parameter<value
//   - parameter>=value
//   - parameter<=value
//   - condition && condition (AND)
//   - condition || condition (OR)
func (ce *ConditionEvaluator) EvaluateCondition(condition string) (bool, error) {
	if condition == "" {
		return false, fmt.Errorf("empty condition")
	}

	return ce.evaluateExpression(condition)
}

// evaluateExpression handles logical operators with proper precedence
// AND (&&) has higher precedence than OR (||)
func (ce *ConditionEvaluator) evaluateExpression(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Split by OR (lower precedence) - but be careful with AND
	orParts := ce.splitByOperator(expr, "||")
	if len(orParts) > 1 {
		// This is an OR expression
		for _, part := range orParts {
			result, err := ce.evaluateExpression(strings.TrimSpace(part))
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// Split by AND (higher precedence)
	andParts := ce.splitByOperator(expr, "&&")
	if len(andParts) > 1 {
		// This is an AND expression
		for _, part := range andParts {
			result, err := ce.evaluateExpression(strings.TrimSpace(part))
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// No logical operators, evaluate comparison
	return ce.evaluateComparison(expr)
}

// splitByOperator splits expression by operator, respecting nested expressions
func (ce *ConditionEvaluator) splitByOperator(expr string, operator string) []string {
	var parts []string
	var current strings.Builder
	i := 0

	for i < len(expr) {
		if i+len(operator) <= len(expr) && expr[i:i+len(operator)] == operator {
			parts = append(parts, current.String())
			current.Reset()
			i += len(operator)
			continue
		}
		current.WriteByte(expr[i])
		i++
	}

	parts = append(parts, current.String())
	return parts
}

// evaluateComparison evaluates a single comparison expression
func (ce *ConditionEvaluator) evaluateComparison(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Try each operator (in order of length to match longest first)
	operators := []string{">=", "<=", "!=", "=", ">", "<"}
	for _, op := range operators {
		if idx := strings.Index(expr, op); idx != -1 {
			return ce.evaluateOperator(expr, op, idx)
		}
	}

	return false, fmt.Errorf("invalid condition syntax: %s", expr)
}

// evaluateOperator evaluates a single operator comparison
func (ce *ConditionEvaluator) evaluateOperator(expr, operator string, opIdx int) (bool, error) {
	paramName := strings.TrimSpace(expr[:opIdx])
	expectedValue := strings.TrimSpace(expr[opIdx+len(operator):])

	// Check if parameter exists
	actualValue, exists := ce.parameters[paramName]
	if !exists {
		// If parameter doesn't exist, check for special case operators
		if operator == "!=" {
			return true, nil // undefined != value is true
		}
		if operator == "=" {
			return false, nil // undefined = value is false
		}
		return false, fmt.Errorf("parameter not found: %s", paramName)
	}

	// Normalize both values for comparison
	actualStr := ce.valueToString(actualValue)

	return ce.compareValues(actualStr, expectedValue, operator)
}

// compareValues compares two string values using the given operator
func (ce *ConditionEvaluator) compareValues(actual, expected, operator string) (bool, error) {
	switch operator {
	case "=":
		return ce.compareEqual(actual, expected), nil

	case "!=":
		return !ce.compareEqual(actual, expected), nil

	case ">":
		return ce.compareGreater(actual, expected)

	case "<":
		return ce.compareLess(actual, expected)

	case ">=":
		return ce.compareGreaterEqual(actual, expected)

	case "<=":
		return ce.compareLessEqual(actual, expected)

	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// compareEqual compares two values for equality with type coercion
func (ce *ConditionEvaluator) compareEqual(actual, expected string) bool {
	// Direct string comparison first
	if actual == expected {
		return true
	}

	// Try boolean comparison
	actualBool := ce.stringToBool(actual)
	expectedBool := ce.stringToBool(expected)
	if actualBool != nil && expectedBool != nil {
		return *actualBool == *expectedBool
	}

	// Try numeric comparison
	actualNum, actualErr := strconv.ParseFloat(actual, 64)
	expectedNum, expectedErr := strconv.ParseFloat(expected, 64)
	if actualErr == nil && expectedErr == nil {
		return actualNum == expectedNum
	}

	return false
}

// compareGreater compares if actual > expected
func (ce *ConditionEvaluator) compareGreater(actual, expected string) (bool, error) {
	actualNum, actualErr := strconv.ParseFloat(actual, 64)
	expectedNum, expectedErr := strconv.ParseFloat(expected, 64)

	if actualErr != nil || expectedErr != nil {
		return false, fmt.Errorf("cannot compare non-numeric values with >: %s > %s", actual, expected)
	}

	return actualNum > expectedNum, nil
}

// compareLess compares if actual < expected
func (ce *ConditionEvaluator) compareLess(actual, expected string) (bool, error) {
	actualNum, actualErr := strconv.ParseFloat(actual, 64)
	expectedNum, expectedErr := strconv.ParseFloat(expected, 64)

	if actualErr != nil || expectedErr != nil {
		return false, fmt.Errorf("cannot compare non-numeric values with <: %s < %s", actual, expected)
	}

	return actualNum < expectedNum, nil
}

// compareGreaterEqual compares if actual >= expected
func (ce *ConditionEvaluator) compareGreaterEqual(actual, expected string) (bool, error) {
	actualNum, actualErr := strconv.ParseFloat(actual, 64)
	expectedNum, expectedErr := strconv.ParseFloat(expected, 64)

	if actualErr != nil || expectedErr != nil {
		return false, fmt.Errorf("cannot compare non-numeric values with >=: %s >= %s", actual, expected)
	}

	return actualNum >= expectedNum, nil
}

// compareLessEqual compares if actual <= expected
func (ce *ConditionEvaluator) compareLessEqual(actual, expected string) (bool, error) {
	actualNum, actualErr := strconv.ParseFloat(actual, 64)
	expectedNum, expectedErr := strconv.ParseFloat(expected, 64)

	if actualErr != nil || expectedErr != nil {
		return false, fmt.Errorf("cannot compare non-numeric values with <=: %s <= %s", actual, expected)
	}

	return actualNum <= expectedNum, nil
}

// valueToString converts any value to string for comparison
func (ce *ConditionEvaluator) valueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return fmt.Sprintf("%v", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%v", v)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// stringToBool converts a string to boolean, returns nil if not a boolean
func (ce *ConditionEvaluator) stringToBool(s string) *bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "yes", "1":
		result := true
		return &result
	case "false", "no", "0":
		result := false
		return &result
	default:
		return nil
	}
}

// ValidateConditionExpression validates condition syntax
func ValidateConditionExpression(condition string) error {
	if condition == "" {
		return fmt.Errorf("condition cannot be empty")
	}

	// Check for balanced logical operators
	if err := validateOperatorBalance(condition); err != nil {
		return err
	}

	// Check that each part is a valid comparison
	evaluator := NewConditionEvaluator(map[string]interface{}{
		"_test": "value",
	})

	// Try to parse the condition structure
	_, err := evaluator.parseConditionStructure(condition)
	return err
}

// validateOperatorBalance checks for valid operator usage
func validateOperatorBalance(condition string) error {
	operators := []string{"&&", "||"}
	for _, op := range operators {
		parts := strings.Split(condition, op)
		if len(parts) > 1 {
			// Check that we don't have empty parts
			for i, part := range parts {
				if strings.TrimSpace(part) == "" {
					if i == 0 || i == len(parts)-1 {
						return fmt.Errorf("operator %s cannot be at start or end", op)
					}
					return fmt.Errorf("operator %s has empty operand", op)
				}
			}
		}
	}
	return nil
}

// parseConditionStructure validates the structure of a condition
func (ce *ConditionEvaluator) parseConditionStructure(condition string) (bool, error) {
	condition = strings.TrimSpace(condition)

	// Handle logical operators
	if strings.Contains(condition, "||") {
		parts := strings.Split(condition, "||")
		for _, part := range parts {
			if _, err := ce.parseConditionStructure(strings.TrimSpace(part)); err != nil {
				return false, err
			}
		}
		return true, nil
	}

	if strings.Contains(condition, "&&") {
		parts := strings.Split(condition, "&&")
		for _, part := range parts {
			if _, err := ce.parseConditionStructure(strings.TrimSpace(part)); err != nil {
				return false, err
			}
		}
		return true, nil
	}

	// Validate comparison syntax
	operators := []string{">=", "<=", "!=", "=", ">", "<"}
	for _, op := range operators {
		if strings.Contains(condition, op) {
			parts := strings.Split(condition, op)
			if len(parts) != 2 {
				return false, fmt.Errorf("invalid comparison syntax: %s", condition)
			}
			paramName := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if paramName == "" {
				return false, fmt.Errorf("missing parameter name in condition: %s", condition)
			}
			if value == "" {
				return false, fmt.Errorf("missing value in condition: %s", condition)
			}

			// Validate parameter name format
			if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(paramName) {
				return false, fmt.Errorf("invalid parameter name: %s", paramName)
			}

			return true, nil
		}
	}

	return false, fmt.Errorf("no valid operator found in condition: %s", condition)
}
