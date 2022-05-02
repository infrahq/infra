package api

type CertificateSigningRequest struct {
	CertificatePEM []byte `json:"certificatePEM" note:"The PEM-encoded certificate to be signed, not including the private key." validate:"required"`
}

type CertificateSigningResponse struct {
	SignedCertificatePEM    []byte `json:"signedCertificatePEM" note:"the pem-encoded signed certificate"`
	CertificateAuthorityPEM []byte `json:"certificateAuthorityPEM" note:"the pem-encoded CA public certificate"`
}
