package icp

import "encoding/asn1"

func idSubjectKeyIdentifier() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{2, 5, 29, 14}
}

func idAuthorityKeyIdentifier() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{2, 5, 29, 35}
}

func idCeBasicConstraints() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{2, 5, 29, 19}
}

func idCeKeyUsage() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{2, 5, 29, 15}
}

type attributeT struct {
	RawContent asn1.RawContent
	Type       asn1.ObjectIdentifier
	Values     []interface{} `asn1:"set"`
}

type extensionT struct {
	RawContent asn1.RawContent
	ExtnID     asn1.ObjectIdentifier
	Critical   bool `asn1:"optional"`
	ExtnValue  []byte
}

type attributeCertificateV1_T struct {
	RawContent         asn1.RawContent
	AcInfo             attributeCertificateInfoV1_T
	SignatureAlgorithm algorithmIdentifierT
	Signature          asn1.BitString
}

type subjectOfAttributeCertificateInfoV1_T struct {
	RawContent        asn1.RawContent
	BaseCertificateID issuerSerialT  `asn1:"tag:0,optional,omitempty"`
	SubjectName       []generalNameT `asn1:"tag:1,optional,omitempty"`
}

type attributeCertificateInfoV1_T struct {
	RawContent            asn1.RawContent
	Version               int
	Subject               subjectOfAttributeCertificateInfoV1_T
	Issuer                []generalNameT
	Signature             algorithmIdentifierT
	SerialNumber          int
	AttCertValidityPeriod generalizedValidityT
	Attributes            []attributeT
	IssuerUniqueID        asn1.BitString `asn1:"optional"`
	Extensions            []extensionT   `asn1:"optional"`
}

// Also known as AttributeCertificate
type attributeCertificateV2_T struct {
	RawContent         asn1.RawContent
	ACInfo             attributeCertificateInfoT
	SignatureAlgorithm algorithmIdentifierT
	SignatureValue     asn1.BitString
}

type attributeCertificateInfoT struct {
	RawContent             asn1.RawContent
	Version                int
	Holder                 holderT
	IssuerV1               []generalNameT `asn1:"optional,omitempty"`
	IssuerV2               v2FormT        `asn1:"optional,omitempty,tag:0"`
	Signature              algorithmIdentifierT
	SerialNumber           int
	AttrCertValidityPeriod generalizedValidityT
	Attributes             []attributeT
	IssuerUniqueID         asn1.BitString `asn1:"optional,omitempty"`
	Extensions             []extensionT   `asn1:"optional,omitempty"`
}

func (acert *attributeCertificateInfoT) SetAppropriateVersion() {
	acert.Version = 1
}

type v2FormT struct {
	RawContent        asn1.RawContent
	IssuerName        []generalNameT    `asn1:"optional,omitempty"`
	BaseCertificateID issuerSerialT     `asn1:"optional,omitempty,tag:0"`
	ObjectDigestInfo  objectDigestInfoT `asn1:"optional,omitempty,tag:1"`
}

type extKeyUsage struct {
	Exists           bool
	DigitalSignature bool
	NonRepudiation   bool
	KeyEncipherment  bool
	DataEncipherment bool
	KeyAgreement     bool
	KeyCertSign      bool
	CRLSign          bool
}

func (ans *extKeyUsage) FromExtensionT(ext extensionT) CodedError {
	seq := asn1.BitString{}
	_, err := asn1.Unmarshal(ext.ExtnValue, &seq)
	if err != nil {
		merr := NewMultiError("failed to parse key usage extention as bit sequence", ERR_PARSE_EXTENSION, nil, err)
		merr.SetParam("raw-data", ext.ExtnValue)
		return merr
	}
	ans.Exists = true
	ans.DigitalSignature = (seq.At(0) != 0)
	ans.NonRepudiation = (seq.At(1) != 0)
	ans.KeyEncipherment = (seq.At(2) != 0)
	ans.DataEncipherment = (seq.At(3) != 0)
	ans.KeyAgreement = (seq.At(4) != 0)
	ans.KeyCertSign = (seq.At(5) != 0)
	ans.CRLSign = (seq.At(6) != 0)
	return nil
}

type extBasicConstraints struct {
	Exists  bool
	CA      bool
	PathLen int
}

// I had to created this struct because encoding/asn1 does can't ignore fields with `asn1:"-"`
type extBasicConstraints_raw struct {
	CA      bool
	PathLen int `asn1:"optional"`
}

func (ans *extBasicConstraints) FromExtensionT(ext extensionT) CodedError {
	raw := extBasicConstraints_raw{}
	_, err := asn1.Unmarshal(ext.ExtnValue, &raw)
	if err != nil {
		merr := NewMultiError("failed to parse basic constraints extention", ERR_PARSE_EXTENSION, nil, err)
		merr.SetParam("raw-data", ext.ExtnValue)
		return merr
	}
	ans.Exists = true
	ans.CA = raw.CA
	ans.PathLen = raw.PathLen
	return nil
}
