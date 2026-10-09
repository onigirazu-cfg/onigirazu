package expression

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SecretLookup reads a field of a secret from a provider (bitwarden,
// vault); the CLI sets it. Without it, templates that use secrets fail.
var SecretLookup func(provider, item, field string) (string, error)

func secretValue(provider, item, field string) (string, error) {
	if SecretLookup == nil {
		return "", fmt.Errorf("secrets are not available here")
	}
	return SecretLookup(provider, item, field)
}

// bitwardenLookup is lookup('community.general.bitwarden', 'item', ...,
// field='password'): one value per item, the password when no field is given
func bitwardenLookup(terms []interface{}) ([]interface{}, error) {
	field := "password"
	var items []string
	for i := 0; i < len(terms); i++ {
		t := fmt.Sprint(terms[i])
		// field='x' arrives as the pair "field", "x"
		if t == "field" && i+1 < len(terms) {
			field = fmt.Sprint(terms[i+1])
			i++
			continue
		}
		items = append(items, t)
	}
	out := make([]interface{}, 0, len(items))
	for _, item := range items {
		v, err := secretValue("bitwarden", item, field)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// lookupArgs splits the terms of a secrets lookup into positional terms
// and the key=value options Ansible's plugins take (they arrive as pairs)
func lookupArgs(terms []interface{}, options ...string) ([]interface{}, map[string]interface{}) {
	known := map[string]bool{}
	for _, o := range options {
		known[o] = true
	}
	kw := map[string]interface{}{}
	var pos []interface{}
	for i := 0; i < len(terms); i++ {
		if s, ok := terms[i].(string); ok && known[s] && i+1 < len(terms) {
			kw[s] = terms[i+1]
			i++
			continue
		}
		pos = append(pos, terms[i])
	}
	return pos, kw
}

// sopsLookup is lookup('community.sops.sops', 'file', rstrip=True): the
// plain text of a SOPS-encrypted file
func sopsLookup(resolve func(string) string, terms []interface{}) ([]interface{}, error) {
	files, kw := lookupArgs(terms, "rstrip", "base64", "input_type", "output_type")
	if SOPSFile == nil {
		return nil, fmt.Errorf("sops files are not available here")
	}
	out := make([]interface{}, 0, len(files))
	for _, f := range files {
		text, err := SOPSFile(resolve(fmt.Sprint(f)))
		if err != nil {
			return nil, err
		}
		if v, set := kw["rstrip"]; !set || Truthy(v) {
			text = strings.TrimRight(text, "\r\n")
		}
		out = append(out, text)
	}
	return out, nil
}

// SOPSFile reads a SOPS-encrypted file as plain text; the CLI sets it
var SOPSFile func(path string) (string, error)

// hashiVaultLookup is lookup('community.hashi_vault.hashi_vault',
// 'secret=secret/data/app/db:password'): the field, or the whole secret as
// a dict; the mount's data/ prefix of KV v2 paths is accepted and dropped
func hashiVaultLookup(terms []interface{}) ([]interface{}, error) {
	pos, kw := lookupArgs(terms, "secret", "url", "auth_method", "token", "namespace", "engine_mount_point")
	var specs []string
	if s, ok := kw["secret"]; ok {
		specs = append(specs, fmt.Sprint(s))
	}
	for _, t := range pos {
		for _, part := range strings.Fields(fmt.Sprint(t)) {
			specs = append(specs, strings.TrimPrefix(part, "secret="))
		}
	}
	out := make([]interface{}, 0, len(specs))
	for _, spec := range specs {
		path, field, _ := strings.Cut(spec, ":")
		v, err := vaultSecret(kvPath(path), field)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// vaultKV2GetLookup is lookup('community.hashi_vault.vault_kv2_get',
// 'app/db'): the secret's data as a dict (under "secret", as the plugin
// returns it, with "data" beside it)
func vaultKV2GetLookup(terms []interface{}) ([]interface{}, error) {
	paths, _ := lookupArgs(terms, "engine_mount_point", "url", "auth_method", "token", "namespace")
	out := make([]interface{}, 0, len(paths))
	for _, p := range paths {
		v, err := vaultSecret(kvPath(fmt.Sprint(p)), "")
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{"secret": v, "data": map[string]interface{}{"data": v}})
	}
	return out, nil
}

// kvPath drops the "<mount>/data/" prefix Ansible's hashi_vault lookup
// takes for KV v2 (the mount comes from the configuration)
func kvPath(path string) string {
	if i := strings.Index(path, "/data/"); i >= 0 {
		return path[i+len("/data/"):]
	}
	return path
}

// vaultSecret is the field of a Vault secret, or the whole secret as a dict
func vaultSecret(path, field string) (interface{}, error) {
	v, err := secretValue("vault", path, field)
	if err != nil {
		return nil, err
	}
	if field != "" {
		return v, nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(v), &data); err != nil {
		return v, nil
	}
	return data, nil
}
