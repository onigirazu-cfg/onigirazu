package expression

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"math/big"
	"strings"
)

// SHA-crypt ($5$ and $6$), as glibc's crypt(3) and Ansible's password_hash
// produce them (https://www.akkadia.org/drepper/SHA-crypt.txt)

const cryptAlphabet = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	sha512Order = [][3]int{{0, 21, 42}, {22, 43, 1}, {44, 2, 23}, {3, 24, 45}, {25, 46, 4}, {47, 5, 26}, {6, 27, 48},
		{28, 49, 7}, {50, 8, 29}, {9, 30, 51}, {31, 52, 10}, {53, 11, 32}, {12, 33, 54}, {34, 55, 13}, {56, 14, 35},
		{15, 36, 57}, {37, 58, 16}, {59, 17, 38}, {18, 39, 60}, {40, 61, 19}, {62, 20, 41}}
	sha256Order = [][3]int{{0, 10, 20}, {21, 1, 11}, {12, 22, 2}, {3, 13, 23}, {24, 4, 14}, {15, 25, 5}, {6, 16, 26},
		{27, 7, 17}, {18, 28, 8}, {9, 19, 29}}
)

// shaCrypt hashes password with salt; rounds 0 means the default 5000,
// which is not written into the result
func shaCrypt(sha512Variant bool, password, salt string, rounds int) string {
	newHash, prefix, order := sha256.New, "$5$", sha256Order
	if sha512Variant {
		newHash, prefix, order = sha512.New, "$6$", sha512Order
	}
	if len(salt) > 16 {
		salt = salt[:16]
	}
	explicit := rounds != 0
	if rounds == 0 {
		rounds = 5000
	}
	rounds = max(1000, min(rounds, 999999999))
	p, s := []byte(password), []byte(salt)
	sum := func(h hash.Hash) []byte { return h.Sum(nil) }

	b := newHash()
	b.Write(p)
	b.Write(s)
	b.Write(p)
	digestB := sum(b)

	a := newHash()
	a.Write(p)
	a.Write(s)
	size := len(digestB)
	for i := len(p); i > 0; i -= size {
		a.Write(digestB[:min(i, size)])
	}
	for i := len(p); i > 0; i >>= 1 {
		if i&1 != 0 {
			a.Write(digestB)
		} else {
			a.Write(p)
		}
	}
	digestA := sum(a)

	dp := newHash()
	for range p {
		dp.Write(p)
	}
	pBytes := repeatTo(sum(dp), len(p))

	ds := newHash()
	for i := 0; i < 16+int(digestA[0]); i++ {
		ds.Write(s)
	}
	sBytes := repeatTo(sum(ds), len(s))

	c := digestA
	for i := 0; i < rounds; i++ {
		h := newHash()
		if i&1 != 0 {
			h.Write(pBytes)
		} else {
			h.Write(c)
		}
		if i%3 != 0 {
			h.Write(sBytes)
		}
		if i%7 != 0 {
			h.Write(pBytes)
		}
		if i&1 != 0 {
			h.Write(c)
		} else {
			h.Write(pBytes)
		}
		c = sum(h)
	}

	var out strings.Builder
	out.WriteString(prefix)
	if explicit {
		fmt.Fprintf(&out, "rounds=%d$", rounds)
	}
	out.WriteString(salt)
	out.WriteByte('$')
	for _, o := range order {
		encode24(&out, c[o[0]], c[o[1]], c[o[2]], 4)
	}
	if sha512Variant {
		encode24(&out, 0, 0, c[63], 2)
	} else {
		encode24(&out, 0, c[31], c[30], 3)
	}
	return out.String()
}

func repeatTo(digest []byte, n int) []byte {
	out := make([]byte, 0, n)
	for len(out) < n {
		out = append(out, digest[:min(len(digest), n-len(out))]...)
	}
	return out
}

func encode24(out *strings.Builder, b2, b1, b0 byte, n int) {
	w := uint(b2)<<16 | uint(b1)<<8 | uint(b0)
	for ; n > 0; n-- {
		out.WriteByte(cryptAlphabet[w&0x3f])
		w >>= 6
	}
}

// randomSalt is 16 characters of the crypt alphabet
func randomSalt() (string, error) {
	var b strings.Builder
	for i := 0; i < 16; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(cryptAlphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(cryptAlphabet[n.Int64()])
	}
	return b.String(), nil
}

// passwordHash is Ansible's password_hash(hashtype, salt, rounds) filter
func passwordHash(p ...interface{}) (interface{}, error) {
	if len(p) == 0 || p[0] == nil {
		return nil, fmt.Errorf("password_hash needs a password")
	}
	kind := "sha512"
	if len(p) > 1 && p[1] != nil {
		kind = fmt.Sprint(p[1])
	}
	salt := ""
	if len(p) > 2 && p[2] != nil {
		salt = fmt.Sprint(p[2])
	}
	rounds := 0
	if len(p) > 3 {
		if n, ok := number(p[3]); ok {
			rounds = int(n)
		}
	}
	if salt == "" {
		var err error
		if salt, err = randomSalt(); err != nil {
			return nil, err
		}
	}
	switch kind {
	case "sha512", "sha512_crypt":
		return shaCrypt(true, fmt.Sprint(p[0]), salt, rounds), nil
	case "sha256", "sha256_crypt":
		return shaCrypt(false, fmt.Sprint(p[0]), salt, rounds), nil
	}
	return nil, fmt.Errorf("password_hash: unsupported hash type %q (sha512, sha256)", kind)
}
