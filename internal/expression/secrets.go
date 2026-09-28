package expression

import "fmt"

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
