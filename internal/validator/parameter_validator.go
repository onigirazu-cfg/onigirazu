package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ParameterValidator handles parameter validation for roles
type ParameterValidator struct {
	parameters      map[string]types.ParameterDef
	customValidator *CustomValidator
}

// NewParameterValidator creates a new validator with a parameter schema
func NewParameterValidator(parameters map[string]types.ParameterDef) *ParameterValidator {
	return &ParameterValidator{
		parameters:      parameters,
		customValidator: NewCustomValidator(),
	}
}

// RegisterCustomValidator registers a new custom validator
func (pv *ParameterValidator) RegisterCustomValidator(name string, validator CustomValidatorFunc) error {
	if pv.customValidator == nil {
		pv.customValidator = NewCustomValidator()
	}
	return pv.customValidator.RegisterValidator(name, validator)
}

// ValidateParameters validates provided parameters against the schema
func (pv *ParameterValidator) ValidateParameters(vars map[string]interface{}) *types.ValidationResult {
	result := &types.ValidationResult{
		Valid:  true,
		Errors: []types.ParameterValidationError{},
	}

	// Check required parameters
	for paramName, paramDef := range pv.parameters {
		isRequired := paramDef.Required
		var conditionError string

		// Check if parameter is conditionally required
		if !isRequired && paramDef.ConditionalRequirement != nil {
			condEval := NewConditionEvaluator(vars)
			condResult, err := condEval.EvaluateCondition(paramDef.ConditionalRequirement.Condition)
			if err == nil && condResult {
				isRequired = true
			}
			if err != nil {
				conditionError = fmt.Sprintf("error evaluating condition: %v", err)
			}
		}

		if isRequired {
			value, exists := vars[paramName]
			if !exists {
				result.Valid = false
				errorMsg := "required parameter missing"
				if paramDef.ConditionalRequirement != nil && paramDef.ConditionalRequirement.ErrorMsg != "" {
					errorMsg = paramDef.ConditionalRequirement.ErrorMsg
				}
				if conditionError != "" {
					errorMsg = conditionError
				}
				result.Errors = append(result.Errors, types.ParameterValidationError{
					Parameter: paramName,
					Error:     errorMsg,
					Value:     nil,
				})
				continue
			}

			// Validate the provided value
			if err := pv.validateParameter(paramName, value, paramDef); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, *err)
			}
		} else if value, exists := vars[paramName]; exists {
			// Validate optional parameters if provided
			if err := pv.validateParameter(paramName, value, paramDef); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, *err)
			}
		}
	}

	return result
}

// validateParameter validates a single parameter
func (pv *ParameterValidator) validateParameter(name string, value interface{}, def types.ParameterDef) *types.ParameterValidationError {
	// First validate type and constraints
	var typeError *types.ParameterValidationError
	switch def.Type {
	case "string":
		typeError = pv.validateString(name, value, def)
	case "integer":
		typeError = pv.validateInteger(name, value, def)
	case "boolean":
		typeError = pv.validateBoolean(name, value)
	case "array":
		typeError = pv.validateArray(name, value, def)
	case "object":
		typeError = pv.validateObject(name, value, def)
	case "float", "number":
		typeError = pv.validateFloat(name, value, def)
	default:
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("unsupported parameter type: %s", def.Type),
			Value:     value,
		}
	}

	// If type validation failed, return the error
	if typeError != nil {
		return typeError
	}

	// Type validation passed, now execute custom validators if any
	if len(def.Validators) > 0 && pv.customValidator != nil {
		isValid, validationErrors := pv.customValidator.ExecuteValidators(value, def.Validators)
		if !isValid && len(validationErrors) > 0 {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     validationErrors[0], // Return first error
				Value:     value,
			}
		}
	}

	return nil
}

// validateString validates a string parameter
func (pv *ParameterValidator) validateString(name string, value interface{}, def types.ParameterDef) *types.ParameterValidationError {
	strVal, ok := value.(string)
	if !ok {
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("expected string, got %T", value),
			Value:     value,
		}
	}

	constraints := def.Constraints

	// Check minimum length
	if constraints.MinLength > 0 && len(strVal) < constraints.MinLength {
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("string length %d is less than minimum %d", len(strVal), constraints.MinLength),
			Value:     value,
		}
	}

	// Check maximum length
	if constraints.MaxLength > 0 && len(strVal) > constraints.MaxLength {
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("string length %d exceeds maximum %d", len(strVal), constraints.MaxLength),
			Value:     value,
		}
	}

	// Check pattern
	if constraints.Pattern != "" {
		if matched, err := regexp.MatchString(constraints.Pattern, strVal); err != nil {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("invalid regex pattern: %v", err),
				Value:     value,
			}
		} else if !matched {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("string does not match pattern: %s", constraints.Pattern),
				Value:     value,
			}
		}
	}

	// Check enum
	if len(constraints.Enum) > 0 {
		found := false
		for _, allowed := range constraints.Enum {
			if allowed == strVal {
				found = true
				break
			}
		}
		if !found {
			allowedStr := fmt.Sprintf("%v", constraints.Enum)
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("value must be one of: %s", allowedStr),
				Value:     value,
			}
		}
	}

	return nil
}

// validateInteger validates an integer parameter
func (pv *ParameterValidator) validateInteger(name string, value interface{}, def types.ParameterDef) *types.ParameterValidationError {
	var intVal int64

	switch v := value.(type) {
	case int:
		intVal = int64(v)
	case int32:
		intVal = int64(v)
	case int64:
		intVal = v
	case float64:
		// Allow float64 if it's a whole number
		if v != float64(int64(v)) {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("expected integer, got float %v", v),
				Value:     value,
			}
		}
		intVal = int64(v)
	case string:
		// Try to parse string as integer
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("cannot parse %q as integer: %v", v, err),
				Value:     value,
			}
		}
		intVal = parsed
	default:
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("expected integer, got %T", value),
			Value:     value,
		}
	}

	constraints := def.Constraints

	// Check minimum
	if constraints.Minimum != nil {
		minVal := pv.toInt64(constraints.Minimum)
		if intVal < minVal {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("value %d is less than minimum %d", intVal, minVal),
				Value:     value,
			}
		}
	}

	// Check maximum
	if constraints.Maximum != nil {
		maxVal := pv.toInt64(constraints.Maximum)
		if intVal > maxVal {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("value %d exceeds maximum %d", intVal, maxVal),
				Value:     value,
			}
		}
	}

	// Check multiple of
	if constraints.MultipleOf != nil {
		multipleVal := pv.toInt64(constraints.MultipleOf)
		if multipleVal != 0 && intVal%multipleVal != 0 {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("value %d must be a multiple of %d", intVal, multipleVal),
				Value:     value,
			}
		}
	}

	// Check enum
	if len(constraints.Enum) > 0 {
		found := false
		for _, allowed := range constraints.Enum {
			if pv.toInt64(allowed) == intVal {
				found = true
				break
			}
		}
		if !found {
			allowedStr := fmt.Sprintf("%v", constraints.Enum)
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("value must be one of: %s", allowedStr),
				Value:     value,
			}
		}
	}

	return nil
}

// validateFloat validates a float parameter
func (pv *ParameterValidator) validateFloat(name string, value interface{}, def types.ParameterDef) *types.ParameterValidationError {
	var floatVal float64

	switch v := value.(type) {
	case float64:
		floatVal = v
	case float32:
		floatVal = float64(v)
	case int:
		floatVal = float64(v)
	case int64:
		floatVal = float64(v)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("cannot parse %q as float: %v", v, err),
				Value:     value,
			}
		}
		floatVal = parsed
	default:
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("expected float, got %T", value),
			Value:     value,
		}
	}

	constraints := def.Constraints

	// Check minimum
	if constraints.Minimum != nil {
		minVal := pv.toFloat64(constraints.Minimum)
		if floatVal < minVal {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("value %.2f is less than minimum %.2f", floatVal, minVal),
				Value:     value,
			}
		}
	}

	// Check maximum
	if constraints.Maximum != nil {
		maxVal := pv.toFloat64(constraints.Maximum)
		if floatVal > maxVal {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("value %.2f exceeds maximum %.2f", floatVal, maxVal),
				Value:     value,
			}
		}
	}

	return nil
}

// validateBoolean validates a boolean parameter
func (pv *ParameterValidator) validateBoolean(name string, value interface{}) *types.ParameterValidationError {
	switch v := value.(type) {
	case bool:
		return nil
	case string:
		// Accept common boolean string representations
		normalized := strings.ToLower(strings.TrimSpace(v))
		if normalized == "true" || normalized == "yes" || normalized == "1" ||
			normalized == "false" || normalized == "no" || normalized == "0" {
			return nil
		}
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("cannot parse %q as boolean (expected: true/yes/1 or false/no/0)", v),
			Value:     value,
		}
	default:
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("expected boolean, got %T", value),
			Value:     value,
		}
	}
}

// validateArray validates an array parameter
func (pv *ParameterValidator) validateArray(name string, value interface{}, def types.ParameterDef) *types.ParameterValidationError {
	arrVal, ok := value.([]interface{})
	if !ok {
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("expected array, got %T", value),
			Value:     value,
		}
	}

	constraints := def.Constraints

	// Check minimum items
	if constraints.MinItems > 0 && len(arrVal) < constraints.MinItems {
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("array has %d items, minimum is %d", len(arrVal), constraints.MinItems),
			Value:     value,
		}
	}

	// Check maximum items
	if constraints.MaxItems > 0 && len(arrVal) > constraints.MaxItems {
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("array has %d items, maximum is %d", len(arrVal), constraints.MaxItems),
			Value:     value,
		}
	}

	// Check unique items
	if constraints.UniqueItems {
		seen := make(map[string]bool)
		for _, item := range arrVal {
			key := fmt.Sprintf("%v", item)
			if seen[key] {
				return &types.ParameterValidationError{
					Parameter: name,
					Error:     fmt.Sprintf("array items must be unique, found duplicate: %v", item),
					Value:     value,
				}
			}
			seen[key] = true
		}
	}

	// Validate item types if specified
	if constraints.ItemsType != "" {
		for i, item := range arrVal {
			itemName := fmt.Sprintf("%s[%d]", name, i)
			itemDef := types.ParameterDef{
				Type:        constraints.ItemsType,
				Constraints: types.ParameterConstraints{},
			}
			if err := pv.validateParameter(itemName, item, itemDef); err != nil {
				err.Parameter = itemName
				err.Error = fmt.Sprintf("item %d: %s", i, err.Error)
				return err
			}
		}
	}

	return nil
}

// validateObject validates an object parameter
func (pv *ParameterValidator) validateObject(name string, value interface{}, def types.ParameterDef) *types.ParameterValidationError {
	objVal, ok := value.(map[string]interface{})
	if !ok {
		return &types.ParameterValidationError{
			Parameter: name,
			Error:     fmt.Sprintf("expected object, got %T", value),
			Value:     value,
		}
	}

	constraints := def.Constraints

	// Check required fields
	for _, requiredField := range constraints.RequiredFields {
		if _, exists := objVal[requiredField]; !exists {
			return &types.ParameterValidationError{
				Parameter: name,
				Error:     fmt.Sprintf("required field missing: %s", requiredField),
				Value:     value,
			}
		}
	}

	return nil
}

// Helper functions for type conversion

// toInt64 converts a value to int64
func (pv *ParameterValidator) toInt64(val interface{}) int64 {
	switch v := val.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// toFloat64 converts a value to float64
func (pv *ParameterValidator) toFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// MergeWithDefaults merges provided variables with default values from schema
func (pv *ParameterValidator) MergeWithDefaults(vars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// First, add all defaults
	for paramName, paramDef := range pv.parameters {
		if paramDef.Default != nil {
			result[paramName] = paramDef.Default
		}
	}

	// Then, override with provided values
	for key, value := range vars {
		result[key] = value
	}

	return result
}

// GetParameterDescription returns a human-readable description of the parameters
func (pv *ParameterValidator) GetParameterDescription() string {
	var sb strings.Builder

	sb.WriteString("Role Parameters:\n")

	for paramName, paramDef := range pv.parameters {
		required := "optional"
		if paramDef.Required {
			required = "required"
		}

		sb.WriteString(fmt.Sprintf("\n  %s (%s) - %s\n", paramName, paramDef.Type, required))

		if paramDef.Description != "" {
			sb.WriteString(fmt.Sprintf("    Description: %s\n", paramDef.Description))
		}

		if paramDef.Default != nil {
			sb.WriteString(fmt.Sprintf("    Default: %v\n", paramDef.Default))
		}

		constraints := paramDef.Constraints
		if constraints.MinLength > 0 {
			sb.WriteString(fmt.Sprintf("    Min length: %d\n", constraints.MinLength))
		}
		if constraints.MaxLength > 0 {
			sb.WriteString(fmt.Sprintf("    Max length: %d\n", constraints.MaxLength))
		}
		if constraints.Minimum != nil {
			sb.WriteString(fmt.Sprintf("    Minimum: %v\n", constraints.Minimum))
		}
		if constraints.Maximum != nil {
			sb.WriteString(fmt.Sprintf("    Maximum: %v\n", constraints.Maximum))
		}
		if constraints.Pattern != "" {
			sb.WriteString(fmt.Sprintf("    Pattern: %s\n", constraints.Pattern))
		}
		if len(constraints.Enum) > 0 {
			sb.WriteString(fmt.Sprintf("    Allowed values: %v\n", constraints.Enum))
		}
	}

	return sb.String()
}

// ValidateCrossParameters validates parameters against cross-parameter rules
func (pv *ParameterValidator) ValidateCrossParameters(vars map[string]interface{}, rules []types.CrossParameterRule) *types.ValidationResult {
	if len(rules) == 0 {
		return &types.ValidationResult{
			Valid:            true,
			Errors:           []types.ParameterValidationError{},
			CrossParamErrors: []types.CrossParameterValidationError{},
		}
	}

	validator := NewCrossParameterValidator(rules)
	return validator.ValidateCrossParameters(vars)
}
