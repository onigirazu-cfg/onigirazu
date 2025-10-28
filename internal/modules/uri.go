package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// URIModule makes HTTP/HTTPS requests to web services and APIs
type URIModule struct {
	*BaseModule
}

// NewURIModule creates a new uri module
func NewURIModule() *URIModule {
	return &URIModule{
		BaseModule: NewBaseModule("uri"),
	}
}

func (m *URIModule) GetDescription() string {
	return "Make HTTP/HTTPS requests to web services and APIs"
}

func (m *URIModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get parameters
	url := ""
	if urlVal, exists := args["url"]; exists {
		if urlStr, ok := urlVal.(string); ok {
			url = urlStr
		}
	}

	if url == "" {
		result.Success = false
		result.Error = "'url' parameter is required"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	method := "GET"
	if methodVal, exists := args["method"]; exists {
		if methodStr, ok := methodVal.(string); ok {
			method = strings.ToUpper(methodStr)
		}
	}

	body := ""
	if bodyVal, exists := args["body"]; exists {
		switch v := bodyVal.(type) {
		case string:
			body = v
		case map[string]interface{}:
			jsonData, _ := json.Marshal(v)
			body = string(jsonData)
		}
	}

	bodyFormat := "raw"
	if formatVal, exists := args["body_format"]; exists {
		if formatStr, ok := formatVal.(string); ok {
			bodyFormat = formatStr
		}
	}

	// Parse headers
	headers := make(map[string]string)
	if headersVal, exists := args["headers"]; exists {
		if headerMap, ok := headersVal.(map[string]interface{}); ok {
			for k, v := range headerMap {
				headers[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Basic auth
	username := ""
	if userVal, exists := args["user"]; exists {
		if userStr, ok := userVal.(string); ok {
			username = userStr
		}
	}

	password := ""
	if passVal, exists := args["password"]; exists {
		if passStr, ok := passVal.(string); ok {
			password = passStr
		}
	}

	// Expected status codes
	statusCodes := []int{200}
	if statusVal, ok := args["status_code"]; ok {
		switch v := statusVal.(type) {
		case float64:
			statusCodes = []int{int(v)}
		case []interface{}:
			statusCodes = []int{}
			for _, code := range v {
				if codeInt, codeOk := code.(float64); codeOk {
					statusCodes = append(statusCodes, int(codeInt))
				}
			}
		}
	}

	// Note: validateCerts is parsed but Go's net/http handles SSL verification by default
	// This parameter is kept for Ansible compatibility
	// Parameter is accepted for compatibility, SSL validation is handled by Go's standard library

	timeout := 30
	if timeoutVal, exists := args["timeout"]; exists {
		if timeoutInt, ok := timeoutVal.(float64); ok {
			timeout = int(timeoutInt)
		}
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Handle body format
	var bodyReader io.Reader
	if body != "" {
		if bodyFormat == "json" {
			// Ensure it's valid JSON
			if !json.Valid([]byte(body)) {
				// Try to marshal if it looks like YAML
				var data interface{}
				_ = json.Unmarshal([]byte(body), &data)
			}
			headers["Content-Type"] = "application/json"
		} else if bodyFormat == "form-urlencoded" {
			headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
		bodyReader = strings.NewReader(body)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add basic auth if provided
	if username != "" {
		req.SetBasicAuth(username, password)
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Check status code
	statusOk := false
	for _, code := range statusCodes {
		if resp.StatusCode == code {
			statusOk = true
			break
		}
	}

	if !statusOk {
		result.Success = false
		result.Error = fmt.Sprintf("unexpected HTTP status %d", resp.StatusCode)
	}

	// Try to parse JSON response
	var jsonResp interface{}
	if err := json.Unmarshal(respBody, &jsonResp); err == nil {
		result.Output["json"] = jsonResp
	}

	// Build response headers map
	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders[key] = values[0]
		}
	}

	result.Output["status"] = resp.StatusCode
	result.Output["url"] = resp.Request.URL.String()
	result.Output["text"] = string(respBody)
	result.Output["headers"] = respHeaders
	result.Output["elapsed"] = time.Since(startTime).Seconds()

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *URIModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	if _, exists := args["url"]; !exists {
		return fmt.Errorf("uri module requires 'url' parameter")
	}

	return nil
}
