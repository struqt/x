package main

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"io"
	"log"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/x25519"
)

var (
	keys    jwk.Set
	payload []byte
)

func main() {
	for i := 0; i < 2; i++ {
		log.Printf("---------- %d %s", i, "JWE A256GCMKW")
		exampleJwe(keys, "201")
		log.Printf("---------- %d %s", i, "JWE RSA_OAEP_256")
		exampleJwe(keys, "202")
		log.Printf("---------- %d %s", i, "JWE ECDH_ES_A256KW")
		exampleJwe(keys, "203") // Y
		log.Printf("---------- %d %s", i, "JWT HS256 A256GCMKW")
		exampleJwt(keys, "201", "101")
		log.Printf("---------- %d %s", i, "JWT EdDSA RSA_OAEP_256")
		exampleJwt(keys, "202", "102")
		log.Printf("---------- %d %s", i, "JWT EdDSA ECDH_ES_A256KW")
		exampleJwt(keys, "203", "102") // Y
		log.Printf("---------- %d %s", i, "JWT ES384 ECDH_ES_A256KW")
		exampleJwt(keys, "203", "103") // Y
	}
}

func init() {
	payload = []byte("This is a secret")
	set := jwk.NewSet()

	var bytes [32]byte
	cnt, err := io.ReadAtLeast(rand.Reader, bytes[:], len(bytes))
	_ = cnt
	if err != nil {
		panic(err)
	}
	var rand256bits = bytes[:]
	if k, err := jwk.FromRaw(rand256bits); err == nil {
		_ = k.Set(jwe.KeyIDKey, "101")
		_ = k.Set(jwe.AlgorithmKey, jwa.HS256) // jwa.SignatureAlgorithm
		_ = set.AddKey(k)
		s0 := base64.StdEncoding.EncodeToString(rand256bits)
		log.Printf("%s %s %s", k.KeyID(), jwa.HS256, s0)
	}
	_, ed25519Key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	if k, err := jwk.FromRaw(ed25519Key); err == nil {
		_ = k.Set(jwe.KeyIDKey, "102")
		_ = k.Set(jwe.AlgorithmKey, jwa.EdDSA) // jwa.SignatureAlgorithm
		_ = set.AddKey(k)
		pem(k, k.KeyID(), jwa.EdDSA.String())
	}
	p384Key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		panic(err)
	}
	if k, err := jwk.FromRaw(p384Key); err == nil {
		_ = k.Set(jwe.KeyIDKey, "103")
		_ = k.Set(jwe.AlgorithmKey, jwa.ES384) // jwa.SignatureAlgorithm
		_ = set.AddKey(k)
		pem(k, k.KeyID(), jwa.ES384.String())
		pem(func() jwk.Key {
			a, _ := k.PublicKey()
			return a
		}(), k.KeyID(), jwa.ES384.String())
	}
	if k, err := jwk.FromRaw(rand256bits); err == nil {
		_ = k.Set(jwe.KeyIDKey, "201")
		_ = k.Set(jwe.AlgorithmKey, jwa.A256GCMKW)       // jwa.KeyEncryptionAlgorithm
		_ = k.Set(jwe.ContentEncryptionKey, jwa.A256GCM) // jwa.ContentEncryptionAlgorithm
		_ = set.AddKey(k)
		s0 := base64.StdEncoding.EncodeToString(rand256bits)
		log.Printf("%s,%s %s", k.KeyID(), jwa.A256GCMKW, s0)
	}
	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Printf("failed to generate private key: %s", err)
		panic(err)
	}
	if k, err := jwk.FromRaw(rsaPrivateKey); err == nil {
		_ = k.Set(jwe.KeyIDKey, "202")
		_ = k.Set(jwe.AlgorithmKey, jwa.RSA_OAEP_256)    // jwa.KeyEncryptionAlgorithm
		_ = k.Set(jwe.ContentEncryptionKey, jwa.A256GCM) // jwa.ContentEncryptionAlgorithm
		_ = set.AddKey(k)
		pem(k, k.KeyID(), jwa.RSA_OAEP_256.String())
	}
	_, x25519Key, err := x25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	if k, err := jwk.FromRaw(x25519Key); err == nil {
		_ = k.Set(jwe.KeyIDKey, "203")
		_ = k.Set(jwe.AlgorithmKey, jwa.ECDH_ES_A256KW)  // jwa.KeyEncryptionAlgorithm
		_ = k.Set(jwe.ContentEncryptionKey, jwa.A256GCM) // jwa.ContentEncryptionAlgorithm
		_ = set.AddKey(k)
		s0 := base64.StdEncoding.EncodeToString(x25519Key)
		log.Printf("%s %s %s", k.KeyID(), jwa.ECDH_ES_A256KW, s0)
	}
	keys = set
}

func pem(k jwk.Key, kid, alg string) {
	if pem, err := jwk.EncodePEM(k); err != nil {
		panic(err)
	} else {
		log.Printf("%s %s\n%s", kid, alg, pem)
		k2, _, x := jwk.DecodePEM(pem)
		if x != nil {
			panic(x)
		}
		_ = k2
		//log.Printf("%s", k2)
	}
}

func decrypt(encrypted []byte) ([]byte, error) {
	log.Printf("%s\n", encrypted)
	log.Println("========== ==========")
	var used jwk.Key
	decrypted, err := jwe.Decrypt(encrypted,
		jwe.WithKeyUsed(&used),
		jwe.WithKeySet(keys, jwe.WithRequireKid(true)),
	)
	if err != nil {
		log.Printf("failed to decrypt: %s", err)
		return nil, err
	}
	log.Printf("%s\n", decrypted)
	return decrypted, nil
}

func exampleJwe(keys jwk.Set, kid string) {
	t0 := time.Now()
	key, ok := keys.LookupKeyID(kid)
	if !ok {
		return
	}
	if pub, err := key.PublicKey(); err == nil {
		key = pub
	}
	var options []jwe.EncryptOption
	options = append(options, jwe.WithKey(key.Algorithm(), key))
	if enc, ok := key.Get(jwe.ContentEncryptionKey); ok {
		if enc, ok := enc.(jwa.ContentEncryptionAlgorithm); ok {
			options = append(options, jwe.WithContentEncryption(enc))
		}
	}
	encrypted, err := jwe.Encrypt(payload, options...)
	if err != nil {
		log.Printf("failed to encrypt payload: %s", err)
		return
	}
	t1 := time.Now()
	log.Printf("enc %s\n", t1.Sub(t0))

	_, _ = decrypt(encrypted)
	t2 := time.Now()
	log.Printf("dec %s\n", t2.Sub(t1))
}

func exampleJwt(keys jwk.Set, kid, skid string) {
	t0 := time.Now()
	signKey, ok := keys.LookupKeyID(skid)
	if !ok {
		return
	}
	key, ok := keys.LookupKeyID(kid)
	if pub, err := key.PublicKey(); err == nil {
		key = pub
	}
	var options []jwt.EncryptOption
	options = append(options, jwt.WithKey(key.Algorithm(), key))
	if enc, ok := key.Get(jwe.ContentEncryptionKey); ok {
		if enc, ok := enc.(jwa.ContentEncryptionAlgorithm); ok {
			options = append(options, jwt.WithEncryptOption(jwe.WithContentEncryption(enc)))
		}
	}
	token, err := jwt.NewBuilder().
		Subject("1234567890").
		Issuer("jwt.example.com").
		IssuedAt(t0).
		Audience([]string{"api.example.com", "web.example.com"}).
		NotBefore(t0).
		Expiration(t0.Add(8*time.Hour)).
		Claim("ext", "test").
		Build()
	if err != nil {
		log.Printf("failed to build: %s", err)
		return
	}
	encrypted, err := jwt.
		NewSerializer().
		Sign(jwt.WithKey(signKey.Algorithm(), signKey)).
		Encrypt(options...).
		Serialize(token)

	if err != nil {
		log.Printf("failed to serialize: %s", err)
		return
	}

	t1 := time.Now()
	log.Printf("enc %s\n", t1.Sub(t0))

	decrypted, _ := decrypt(encrypted)

	t2 := time.Now()
	log.Printf("dec %s\n", t2.Sub(t1))

	if signPub, err := signKey.PublicKey(); err == nil {
		signKey = signPub
	}
	verified, err := jwt.Parse(decrypted, jwt.WithKey(signKey.Algorithm(), signKey))
	if err != nil {
		log.Printf("failed to parse: %s", err)
		return
	}
	log.Printf("%s\n", verified.Audience())
	t3 := time.Now()
	log.Printf("parse %s\n", t3.Sub(t2))
}
