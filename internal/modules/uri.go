package modules

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
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
		TaskName:  taskName(args),
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
		case []interface{}:
			statusCodes = []int{}
			for _, code := range v {
				if codeInt, codeOk := toInt(code); codeOk {
					statusCodes = append(statusCodes, codeInt)
				}
			}
		default:
			if c, ok := toInt(v); ok {
				statusCodes = []int{c}
			}
		}
	}

	// Note: validateCerts is parsed but Go's net/http handles SSL verification by default
	// This parameter is kept for Ansible compatibility
	// Parameter is accepted for compatibility, SSL validation is handled by Go's standard library

	timeout := getIntArg(args, "timeout", 30)

	if body != "" {
		if bodyFormat == "json" {
			headers["Content-Type"] = "application/json"
		} else if bodyFormat == "form-urlencoded" {
			headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}

	// The request is made from the target host with curl, like Ansible's uri
	curl := []string{"curl", "-sS", "-X", shellQuote(method), "--max-time", fmt.Sprint(timeout),
		"-o", `"$b"`, "-D", `"$h"`, "-w", "'%{http_code}'"}
	for key, value := range headers {
		curl = append(curl, "-H", shellQuote(key+": "+value))
	}
	credFile := ""
	if username != "" {
		// Credentials go through a 0600 file, never the command line (ps)
		credFile = fmt.Sprintf("/tmp/.onigirazu-uri-%d", time.Now().UnixNano())
		cfg := fmt.Sprintf("user = \"%s:%s\"\n", curlConfigEscape(username), curlConfigEscape(password))
		if err := putPrivateFile(ctx, host, credFile, []byte(cfg)); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to prepare credentials: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		curl = append(curl, "-K", shellQuote(credFile))
	}
	input := ""
	if body != "" {
		curl = append(curl, "--data-binary", "@-")
		input = "printf '%s' " + shellQuote(body) + " | "
	}
	curl = append(curl, shellQuote(url))
	script := fmt.Sprintf(`b=$(mktemp); h=$(mktemp); trap 'rm -f "$b" "$h" %s' EXIT
code=$(%s%s) || exit $?
printf '%%s\n' "$code"; base64 < "$h" | tr -d '\n'; echo; base64 < "$b" | tr -d '\n'`,
		shellQuote(credFile), input, strings.Join(curl, " "))
	out, err := runShellOnHost(ctx, host, args, script)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	lines := strings.SplitN(strings.TrimRight(out, "\n"), "\n", 3)
	for len(lines) < 3 {
		lines = append(lines, "")
	}
	statusCode, _ := strconv.Atoi(strings.TrimSpace(lines[0]))
	rawHeaders, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[1]))
	respBody, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[2]))

	// Check status code
	statusOk := false
	for _, code := range statusCodes {
		if statusCode == code {
			statusOk = true
			break
		}
	}

	if !statusOk {
		result.Success = false
		result.Error = fmt.Sprintf("unexpected HTTP status %d", statusCode)
	}

	// Try to parse JSON response
	var jsonResp interface{}
	if err := json.Unmarshal(respBody, &jsonResp); err == nil {
		result.Output["json"] = jsonResp
	}

	// Header lines of the (last) response
	respHeaders := make(map[string]string)
	for _, line := range strings.Split(string(rawHeaders), "\n") {
		if k, v, ok := strings.Cut(strings.TrimRight(line, "\r"), ":"); ok && !strings.HasPrefix(k, "HTTP/") {
			respHeaders[http.CanonicalHeaderKey(strings.TrimSpace(k))] = strings.TrimSpace(v)
		}
	}

	result.Output["status"] = statusCode
	result.Output["url"] = url
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

// curlConfigEscape escapes a value for a double-quoted curl config string
func curlConfigEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}

// putPrivateFile creates a 0600 file on the host without passing its content
// on a command line: SFTP for remote hosts, the local filesystem otherwise
func putPrivateFile(ctx context.Context, host types.Host, path string, data []byte) error {
	if sshpkg.IsLocal(host) {
		return os.WriteFile(path, data, 0600)
	}
	pool := sshpkg.GetGlobalPool()
	client, err := pool.GetConnection(host)
	if err != nil {
		return err
	}
	defer pool.ReleaseConnection(host)
	return client.WriteFile(path, data, 0600)
}
