package icp

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/gjvnq/asn1"
)

const VERSION_MAJOR = 0
const VERSION_MINOR = 0
const VERSION_PATCH = 1

// Returns this library version as a string
func Version() string {
	return fmt.Sprintf("%d.%d.%d", VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH)
}

// Outputs a byte sequence as pairs of hexadecimal digits separated by colons. Ex: AA:FF:1E
func NiceHex(buf []byte) string {
	ans := ""
	for i := 0; i < len(buf); i++ {
		if i != 0 {
			ans += ":"
		}
		ans += fmt.Sprintf("%X", buf[i:i+1])
	}
	return ans
}

type ContentInfo struct {
	RawContent  asn1.RawContent
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue
}

type Signable interface {
	GetRawContent() asn1.RawContent
	GetSignatureAlgorithm() AlgorithmIdentifier
	GetSignature() asn1.BitString
}

func GetHasher(alg_id AlgorithmIdentifier) (hash.Hash, crypto.Hash, CodedError) {
	// Check algorithm
	alg := alg_id.Algorithm
	var hasher hash.Hash
	var hash_alg crypto.Hash
	switch {
	case alg.Equal(IdSha1WithRSAEncryption()):
		hasher = sha1.New()
		hash_alg = crypto.SHA1
	case alg.Equal(IdSha256WithRSAEncryption()):
		hasher = sha256.New()
		hash_alg = crypto.SHA256
	case alg.Equal(IdSha384WithRSAEncryption()):
		hasher = sha512.New384()
		hash_alg = crypto.SHA384
	case alg.Equal(IdSha512WithRSAEncryption()):
		hasher = sha512.New()
		hash_alg = crypto.SHA512
	default:
		merr := NewMultiError("unknown algorithm", ERR_UNKOWN_ALGORITHM, nil)
		merr.SetParam("algorithm", alg)
		return nil, crypto.Hash(0), merr
	}
	return hasher, hash_alg, nil
}

func RunHash(hasher hash.Hash, data []byte) []byte {
	hasher.Write(data)
	return hasher.Sum(nil)
}

func RunHashWithReader(hasher hash.Hash, input io.Reader) ([]byte, CodedError) {
	_, err := io.Copy(hasher, input)
	if err != nil {
		return nil, NewMultiError("failed to hash file", ERR_FAILED_HASH, nil)
	}
	return hasher.Sum(nil), nil
}

func VerifySignaure(object Signable, pubkey rsa.PublicKey) CodedError {
	// Check algorithm
	tbs_hasher, tbs_hash_alg, merr := GetHasher(object.GetSignatureAlgorithm())
	if merr != nil {
		return merr
	}

	// Write raw value
	hash_ans := RunHash(tbs_hasher, object.GetRawContent())

	// Verify signature
	sig := object.GetSignature().Bytes
	err := rsa.VerifyPKCS1v15(&pubkey, tbs_hash_alg, hash_ans, sig)
	if err != nil {
		return NewMultiError("failed to verify signature", ERR_BAD_SIGNATURE, nil, err)
	}
	return nil
}

func HTTPGet(url string) ([]byte, int64, CodedError) {
	// Get the data
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		merr := NewMultiError("failed to use GET method", ERR_HTTP, nil, err)
		merr.SetParam("URL", url)
		return nil, 0, merr
	}
	raw, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		merr := NewMultiError("failed to read http response", ERR_HTTP, nil, err)
		merr.SetParam("URL", url)
		return nil, 0, merr
	}

	return raw, resp.ContentLength, nil
}

const ObjectDigestInfo_PublicKey = 0
const ObjectDigestInfo_PublicKeyCert = 1
const ObjectDigestInfo_OtherObjectTypes = 2

type ObjectDigestInfo struct {
	RawContent         asn1.RawContent
	DigestedObjectType int
	OtherObjectTypeID  asn1.ObjectIdentifier `asn1:"optional,omitempty"`
	DigestAlgorithm    AlgorithmIdentifier
	ObjectDigest       asn1.BitString
}

// Also unmarshals UTCTime
type GeneralizedValidity struct {
	RawContent    asn1.RawContent
	NotBeforeTime time.Time `asn1:"generalized"`
	NotAfterTime  time.Time `asn1:"generalized"`
}

func IsZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}