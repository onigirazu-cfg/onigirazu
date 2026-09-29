package parser

import "testing"

func TestAnsibleInventoryKeepsDocumentOrder(t *testing.T) {
	data := []byte(`all:
  hosts:
    zeta:
  children:
    odd:
      hosts:
        h1:
        h3:
    even:
      hosts:
        h2:
`)
	p := NewInventoryParser(&mockLogger{})
	inv, err := p.parseAnsibleYamlInventory(data)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, h := range inv.Hosts {
		got = append(got, h.Name)
	}
	want := []string{"zeta", "h1", "h3", "h2"}
	if len(got) != len(want) {
		t.Fatalf("hosts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hosts = %v, want %v (Ansible's inventory order)", got, want)
		}
	}
}
