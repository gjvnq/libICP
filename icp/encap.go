package icp

import (
	"os"
	"path/filepath"

	"github.com/gjvnq/asn1"
)

type EncapsulatedContentInfo struct {
	RawContent    asn1.RawContent
	EContentType  asn1.ObjectIdentifier
	EContent      []byte `asn1:"explicit,optional,omitempty"`
	fallback_file string
	hashes        map[string][]byte
}

func (ec *EncapsulatedContentInfo) SetFallbackFile(path string) CodedError {
	abs_path, err := filepath.Abs(path)
	if err != nil {
		merr := NewMultiError("failed to get absolute path", ERR_FAILED_ABS_PATH, nil)
		merr.SetParam("path", path)
		return merr
	}

	if _, err := os.Stat(abs_path); os.IsNotExist(err) {
		merr := NewMultiError("fallback file does not exist", ERR_FILE_NOT_EXISTS, nil)
		merr.SetParam("path", abs_path)
		return merr
	}
	ec.fallback_file = abs_path
	return nil
}

func (ec EncapsulatedContentInfo) IsDetached() bool {
	return ec.EContent == nil
}

// Return true if EContent is not nil or if fallback_file exists
func (ec EncapsulatedContentInfo) IsHashable() bool {
	return ec.EContent != nil || ec.fallback_file != ""
}

func (ec *EncapsulatedContentInfo) HashAs(alg_id AlgorithmIdentifier) ([]byte, CodedError) {
	if ec.hashes == nil {
		ec.hashes = make(map[string][]byte)
	}
	// Check if the hash was already calculated
	if ans, ok := ec.hashes[alg_id.ToHex()]; ok {
		return ans, nil
	}
	// Get hasher
	hasher, _, cerr := GetHasher(alg_id)
	if cerr != nil {
		return nil, cerr
	}
	// Hash
	if ec.IsDetached() {
		// Open file
		f, err := os.Open(ec.fallback_file)
		defer f.Close()
		if err != nil {
			merr := NewMultiError("failed to open file", ERR_FAILED_TO_OPEN_FILE, nil)
			merr.SetParam("path", ec.fallback_file)
			return nil, merr
		}
		ans, cerr := RunHashWithReader(hasher, f)
		if cerr != nil {
			return nil, cerr
		}
		ec.hashes[alg_id.ToHex()] = ans
		return ans, nil
	}
	ans := RunHash(hasher, ec.EContent)
	ec.hashes[alg_id.ToHex()] = ans
	return ans, nil
}

/*	According to RFC 5652 Section 5.2 Page 11 Paragraph 2:

	In the degenerate case where there are no signers, the
	EncapsulatedContentInfo value being "signed" is irrelevant.  In this
	case, the content type within the EncapsulatedContentInfo value being
	"signed" MUST be id-data (as defined in Section 4), and the content
	field of the EncapsulatedContentInfo value MUST be omitted.
*/
func (ec *EncapsulatedContentInfo) AdjustForNoSigners() {
	ec.EContentType = IdData()
	ec.EContent = nil
}
