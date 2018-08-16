package libICP

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"math/big"
	"unicode/utf16"

	"github.com/gjvnq/asn1"
)

type pfx_raw struct {
	RawContent asn1.RawContent
	Version    int
	AuthSafe   content_info
	MacData    mac_data `asn1:"optional,omitempty"`
}

func (pfx *pfx_raw) Marshal(password string, cert certificate_pack, key *rsa.PrivateKey) CodedError {
	// Basics
	pfx.Version = 3
	pfx.AuthSafe.ContentType = idData
	safe := make([]safe_bag_octet, 1)
	pfx.AuthSafe.Content = safe

	// Encode private key
	enc_key_bag := encrypted_private_key_info{}
	cerr := enc_key_bag.SetKey(key, password)
	if cerr != nil {
		return cerr
	}
	safe[0].BagId = idData
	key_safe := make(safe_contents, 1)
	key_safe[0].BagId = idPKCS12_8ShroudedKeyBag
	key_safe[0].BagValue = enc_key_bag
	safe[0].BagValue = key_safe

	// Encode certificate
	// safe[1].BagId = idEncryptedData
	// safe[0].BagValue

	// Final encoding
	dat, err := asn1.Marshal(pfx)
	if err != nil {
		return NewMultiError("failed to marshal pfx_raw", ERR_FAILED_TO_ENCODE, nil, err)
	}
	pfx.RawContent = asn1.RawContent(dat)
	return nil
}

type mac_data struct {
	Mac        object_digest_info
	MacSalt    []byte
	Iterations int `asn1:"optional,default:1"`
}

// ASN.1 definition:
//     AuthenticatedSafe ::= SEQUENCE OF ContentInfo
//     -- Data if unencrypted
//     -- EncryptedData if password-encrypted
//     -- EnvelopedData if public key-encrypted
type authenticated_safe []content_info

type safe_contents []safe_bag

type safe_bag_octet struct {
	RawContent asn1.RawContent
	BagId      asn1.ObjectIdentifier
	BagValue   interface{} `asn1:"explicit,tag:0,octet"`
	BagAttr    []attribute `asn1:"set,optional,omitempty"`
}

type safe_bag struct {
	RawContent asn1.RawContent
	BagId      asn1.ObjectIdentifier
	BagValue   interface{} `asn1:"explicit,tag:0"`
	BagAttr    []attribute `asn1:"set,optional,omitempty"`
}

// A KeyBag is a PKCS #8 PrivateKeyInfo. Note that a KeyBag contains only one private key. (OID: pkcs-12 10 1 1)
//     KeyBag ::= PrivateKeyInfo
type private_key_info struct {
	RawContent          asn1.RawContent
	Version             int
	PrivateKeyAlgorithm algorithm_identifier
	PrivateKey          []byte
	Attributes          []attribute `asn1:"tag:0,implicit,set"`
}

func (s *private_key_info) SetKey(key *rsa.PrivateKey) CodedError {
	var cerr CodedError

	s.Version = 0
	s.Attributes = nil
	s.PrivateKeyAlgorithm = algorithm_identifier{}
	s.PrivateKeyAlgorithm.Algorithm = idRSAEncryption
	s.PrivateKey, cerr = marshal_rsa_private_key(key)
	if cerr != nil {
		return cerr
	}

	dat, err := asn1.Marshal(s)
	if err != nil {
		return NewMultiError("failed to marshal private_key_info", ERR_FAILED_TO_ENCODE, nil, err)
	}

	s.RawContent = asn1.RawContent(dat)
	return nil
}

type encrypted_private_key_info struct {
	RawContent asn1.RawContent
	Alg        algorithm_identifier
	Data       []byte
}

func (s *encrypted_private_key_info) SetKey(priv *rsa.PrivateKey, password string) CodedError {
	info := private_key_info{}
	cerr := info.SetKey(priv)
	if cerr != nil {
		return cerr
	}
	return s.SetData(info.RawContent, password)
}

// BUG(x): It supports only idPbeWithSHAAnd3KeyTripleDES_CBC with SHA1
func (s *encrypted_private_key_info) SetData(msg []byte, password string) CodedError {
	param := pbes1_parameters{}

	// Generate salt
	param.Salt = make([]byte, 8)
	_, err := rand.Read(param.Salt)
	if err != nil {
		return NewMultiError("faield to generate random salt", ERR_SECURE_RANDOM, nil, err)
	}
	param.Iterations = 2048

	// Set basics
	s.Alg.Algorithm = idPbeWithSHAAnd3KeyTripleDES_CBC
	s.Alg.Parameters = make([]interface{}, 2)
	s.Alg.Parameters[0] = param.Salt
	s.Alg.Parameters[1] = param.Iterations

	// Convert password
	byte_password := conv_password(password)

	// Generate key
	k := pbkdf(sha1Sum, 20, 64, param.Salt, byte_password, param.Iterations, 1, 24)

	// Generate IV
	iv := pbkdf(sha1Sum, 20, 64, param.Salt, byte_password, param.Iterations, 2, 8)

	// Encrypt
	block, err := des.NewTripleDESCipher(k)
	if err != nil {
		return NewMultiError("faield to open block cipher for triple DES", ERR_FAILED_TO_ENCODE, nil, err)
	}
	block_mode := cipher.NewCBCEncrypter(block, iv)
	paded_msg := pad_msg(msg)
	s.Data = make([]byte, len(paded_msg))
	block_mode.CryptBlocks(s.Data, paded_msg)

	// Encode
	final, err := asn1.Marshal(s)
	if err != nil {
		return NewMultiError("failed to marshal encrypted_private_key_info", ERR_FAILED_TO_ENCODE, nil, err)
	}

	s.RawContent = asn1.RawContent(final)
	return nil
}

// Code taken from github.com/golang/crypto
func sha1Sum(in []byte) []byte {
	sum := sha1.Sum(in)
	return sum[:]
}

// Code taken from github.com/golang/crypto
func fillWithRepeats(pattern []byte, v int) []byte {
	if len(pattern) == 0 {
		return nil
	}
	outputLen := v * ((len(pattern) + v - 1) / v)
	return bytes.Repeat(pattern, (outputLen+len(pattern)-1)/len(pattern))[:outputLen]
}

// Code taken from github.com/golang/crypto
func pbkdf(hash func([]byte) []byte, u, v int, salt, password []byte, r int, ID byte, size int) (key []byte) {
	one := big.NewInt(1)
	// implementation of https://tools.ietf.org/html/rfc7292#appendix-B.2 , RFC text verbatim in comments

	//    Let H be a hash function built around a compression function f:

	//       Z_2^u x Z_2^v -> Z_2^u

	//    (that is, H has a chaining variable and output of length u bits, and
	//    the message input to the compression function of H is v bits).  The
	//    values for u and v are as follows:

	//            HASH FUNCTION     VALUE u        VALUE v
	//              MD2, MD5          128            512
	//                SHA-1           160            512
	//               SHA-224          224            512
	//               SHA-256          256            512
	//               SHA-384          384            1024
	//               SHA-512          512            1024
	//             SHA-512/224        224            1024
	//             SHA-512/256        256            1024

	//    Furthermore, let r be the iteration count.

	//    We assume here that u and v are both multiples of 8, as are the
	//    lengths of the password and salt strings (which we denote by p and s,
	//    respectively) and the number n of pseudorandom bits required.  In
	//    addition, u and v are of course non-zero.

	//    For information on security considerations for MD5 [19], see [25] and
	//    [1], and on those for MD2, see [18].

	//    The following procedure can be used to produce pseudorandom bits for
	//    a particular "purpose" that is identified by a byte called "ID".
	//    This standard specifies 3 different values for the ID byte:

	//    1.  If ID=1, then the pseudorandom bits being produced are to be used
	//        as key material for performing encryption or decryption.

	//    2.  If ID=2, then the pseudorandom bits being produced are to be used
	//        as an IV (Initial Value) for encryption or decryption.

	//    3.  If ID=3, then the pseudorandom bits being produced are to be used
	//        as an integrity key for MACing.

	//    1.  Construct a string, D (the "diversifier"), by concatenating v/8
	//        copies of ID.
	var D []byte
	for i := 0; i < v; i++ {
		D = append(D, ID)
	}

	//    2.  Concatenate copies of the salt together to create a string S of
	//        length v(ceiling(s/v)) bits (the final copy of the salt may be
	//        truncated to create S).  Note that if the salt is the empty
	//        string, then so is S.

	S := fillWithRepeats(salt, v)

	//    3.  Concatenate copies of the password together to create a string P
	//        of length v(ceiling(p/v)) bits (the final copy of the password
	//        may be truncated to create P).  Note that if the password is the
	//        empty string, then so is P.

	P := fillWithRepeats(password, v)

	//    4.  Set I=S||P to be the concatenation of S and P.
	I := append(S, P...)

	//    5.  Set c=ceiling(n/u).
	c := (size + u - 1) / u

	//    6.  For i=1, 2, ..., c, do the following:
	A := make([]byte, c*20)
	var IjBuf []byte
	for i := 0; i < c; i++ {
		//        A.  Set A2=H^r(D||I). (i.e., the r-th hash of D||1,
		//            H(H(H(... H(D||I))))
		Ai := hash(append(D, I...))
		for j := 1; j < r; j++ {
			Ai = hash(Ai)
		}
		copy(A[i*20:], Ai[:])

		if i < c-1 { // skip on last iteration
			// B.  Concatenate copies of Ai to create a string B of length v
			//     bits (the final copy of Ai may be truncated to create B).
			var B []byte
			for len(B) < v {
				B = append(B, Ai[:]...)
			}
			B = B[:v]

			// C.  Treating I as a concatenation I_0, I_1, ..., I_(k-1) of v-bit
			//     blocks, where k=ceiling(s/v)+ceiling(p/v), modify I by
			//     setting I_j=(I_j+B+1) mod 2^v for each j.
			{
				Bbi := new(big.Int).SetBytes(B)
				Ij := new(big.Int)

				for j := 0; j < len(I)/v; j++ {
					Ij.SetBytes(I[j*v : (j+1)*v])
					Ij.Add(Ij, Bbi)
					Ij.Add(Ij, one)
					Ijb := Ij.Bytes()
					// We expect Ijb to be exactly v bytes,
					// if it is longer or shorter we must
					// adjust it accordingly.
					if len(Ijb) > v {
						Ijb = Ijb[len(Ijb)-v:]
					}
					if len(Ijb) < v {
						if IjBuf == nil {
							IjBuf = make([]byte, v)
						}
						bytesShort := v - len(Ijb)
						for i := 0; i < bytesShort; i++ {
							IjBuf[i] = 0
						}
						copy(IjBuf[bytesShort:], Ijb)
						Ijb = IjBuf
					}
					copy(I[j*v:(j+1)*v], Ijb)
				}
			}
		}
	}
	//    7.  Concatenate A_1, A_2, ..., A_c together to form a pseudorandom
	//        bit string, A.

	//    8.  Use the first n bits of A as the output of this entire process.
	return A[:size]

	//    If the above process is being used to generate a DES key, the process
	//    should be used to create 64 random bits, and the key's parity bits
	//    should be set after the 64 bits have been produced.  Similar concerns
	//    hold for 2-key and 3-key triple-DES keys, for CDMF keys, and for any
	//    similar keys with parity bits "built into them".
}

func power_sha1(base []byte, r int) []byte {
	hasher := sha1.New()
	ans := hasher.Sum(base)
	for i := 1; i < r; i++ {
		ans = hasher.Sum(ans)
	}
	return ans
}

func pad_msg(msg []byte) []byte {
	ps := make([]byte, 8-(len(msg)%8))
	for i := 0; i < len(ps); i++ {
		ps[i] = byte(len(ps))
	}
	return append(msg, ps...)
}

func conv_password(password string) []byte {
	runes := []rune(password)
	if len(runes) > 0 && runes[len(runes)-1] != 0 {
		runes = append(runes, 0)
	}

	seq := utf16.Encode(runes)
	passwd := make([]byte, 2*len(seq))
	for i, _ := range seq {
		binary.BigEndian.PutUint16(passwd[2*i:], seq[i])
	}

	return passwd
}

func unencrypt_idPbeWithSHAAnd3KeyTripleDES_CBC(password []byte, iterations int, salt []byte, enc_msg []byte) ([]byte, error) {

	// Derive our key
	k := pbkdf(sha1Sum, 20, 64, salt, password, iterations, 1, 24)

	// Derive the IV
	iv := pbkdf(sha1Sum, 20, 64, salt, password, iterations, 2, 8)

	// Decrypt
	block, err := des.NewTripleDESCipher(k)
	if err != nil {
		return nil, NewMultiError("faield to open block cipher for triple DES", ERR_FAILED_TO_DECODE, nil, err)
	}
	block_mode := cipher.NewCBCDecrypter(block, iv)
	ans := make([]byte, len(enc_msg))
	block_mode.CryptBlocks(ans, enc_msg)
	return ans, nil
}
