package lab

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/xjiang77/rubickx/network-security/internal/evidence"
)

type localCA struct {
	certificate *x509.Certificate
	privateKey  ed25519.PrivateKey
}

func newLocalCA(commonName string, now time.Time) (localCA, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return localCA{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return localCA{}, err
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		return localCA{}, err
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		return localCA{}, err
	}
	return localCA{certificate: certificate, privateKey: privateKey}, nil
}

func (ca localCA) issue(commonName string, usages []x509.ExtKeyUsage, notBefore, notAfter time.Time) (tls.Certificate, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return tls.Certificate{}, err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  usages,
	}
	if commonName == "localhost" {
		template.DNSNames = []string{"localhost"}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.certificate, publicKey, ca.privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der, ca.certificate.Raw}, PrivateKey: privateKey, Leaf: leaf}, nil
}

func certPool(certificates ...*x509.Certificate) *x509.CertPool {
	pool := x509.NewCertPool()
	for _, certificate := range certificates {
		pool.AddCert(certificate)
	}
	return pool
}

func startMTLSServer(serverCertificate tls.Certificate, clientRoots *x509.CertPool) (*httptest.Server, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-TLS-Version", tls.VersionName(request.TLS.Version))
		writer.Header().Set("X-Client-Cert", request.TLS.PeerCertificates[0].Subject.CommonName)
		_, _ = io.WriteString(writer, "mtls-ok")
	})
	server := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler, ReadHeaderTimeout: 2 * time.Second},
	}
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCertificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientRoots,
		MinVersion:   tls.VersionTLS13,
	}
	server.StartTLS()
	return server, nil
}

func tlsGet(ctx context.Context, rawURL, serverName string, roots *x509.CertPool, clientCertificate *tls.Certificate) error {
	config := &tls.Config{RootCAs: roots, ServerName: serverName, MinVersion: tls.VersionTLS13}
	if clientCertificate != nil {
		config.Certificates = []tls.Certificate{*clientCertificate}
	}
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: config}, Timeout: time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK || string(body) != "mtls-ok" || response.Header.Get("X-TLS-Version") != "TLS 1.3" {
		return fmt.Errorf("unexpected mTLS response status=%d body=%q version=%q", response.StatusCode, body, response.Header.Get("X-TLS-Version"))
	}
	return nil
}

func runLab03(ctx context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-03"
	now := time.Now()
	caV1, err := newLocalCA("netsec-ca-v1", now)
	if err != nil {
		return evidence.LabResult{}, err
	}
	serverV1, err := caV1.issue("localhost", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, now.Add(-time.Minute), now.Add(20*time.Minute))
	if err != nil {
		return evidence.LabResult{}, err
	}
	clientV1, err := caV1.issue("client-v1", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, now.Add(-time.Minute), now.Add(20*time.Minute))
	if err != nil {
		return evidence.LabResult{}, err
	}
	serviceV1, err := startMTLSServer(serverV1, certPool(caV1.certificate))
	if err != nil {
		return evidence.LabResult{}, err
	}
	defer serviceV1.Close()
	if err := tlsGet(ctx, serviceV1.URL, "localhost", certPool(caV1.certificate), &clientV1); err != nil {
		return evidence.LabResult{}, fmt.Errorf("normal mTLS: %w", err)
	}
	normal := scenarioEvent(id, evidence.Normal, "lab03-mtls-tls13", "tls_handshake", "tls-client", "handshake_state", "tls=1.3; server_san=localhost; client_cert=client-v1", "allow", "TLS 1.3 verifies the server SAN and the server verifies a client certificate from the configured CA")

	if err := tlsGet(ctx, serviceV1.URL, "wrong.invalid", certPool(caV1.certificate), &clientV1); err == nil {
		return evidence.LabResult{}, fmt.Errorf("SAN mismatch unexpectedly succeeded")
	}
	if _, err := serverV1.Leaf.Verify(x509.VerifyOptions{Roots: certPool(caV1.certificate), DNSName: "localhost", CurrentTime: serverV1.Leaf.NotAfter.Add(time.Second)}); err == nil {
		return evidence.LabResult{}, fmt.Errorf("expired certificate unexpectedly verified")
	}
	reject := scenarioEvent(id, evidence.Reject, "lab03-san-expiry-reject", "certificate_validation", "x509-verifier", "verification_errors", "san_mismatch=rejected; expired_at_not_after_plus_1s=rejected", "deny", "hostname mismatch and certificate expiry both fail before application data is trusted")

	if err := tlsGet(ctx, serviceV1.URL, "localhost", x509.NewCertPool(), &clientV1); err == nil {
		return evidence.LabResult{}, fmt.Errorf("unknown server CA unexpectedly succeeded")
	}
	if err := tlsGet(ctx, serviceV1.URL, "localhost", certPool(caV1.certificate), nil); err == nil {
		return evidence.LabResult{}, fmt.Errorf("missing client certificate unexpectedly succeeded")
	}
	dependency := scenarioEvent(id, evidence.DependencyFailure, "lab03-trust-dependency-failure", "tls_handshake", "tls-client", "handshake_errors", "unknown_server_ca=rejected; missing_client_cert=rejected", "deny", "missing trust material on either side makes the mTLS handshake fail closed")

	caV2, err := newLocalCA("netsec-ca-v2", now)
	if err != nil {
		return evidence.LabResult{}, err
	}
	serverV2, err := caV2.issue("localhost", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, now.Add(-time.Minute), now.Add(20*time.Minute))
	if err != nil {
		return evidence.LabResult{}, err
	}
	clientV2, err := caV2.issue("client-v2", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, now.Add(-time.Minute), now.Add(20*time.Minute))
	if err != nil {
		return evidence.LabResult{}, err
	}
	serviceV2, err := startMTLSServer(serverV2, certPool(caV1.certificate, caV2.certificate))
	if err != nil {
		return evidence.LabResult{}, err
	}
	defer serviceV2.Close()
	overlapRoots := certPool(caV1.certificate, caV2.certificate)
	if err := tlsGet(ctx, serviceV1.URL, "localhost", overlapRoots, &clientV1); err != nil {
		return evidence.LabResult{}, fmt.Errorf("old trust during overlap: %w", err)
	}
	if err := tlsGet(ctx, serviceV2.URL, "localhost", overlapRoots, &clientV2); err != nil {
		return evidence.LabResult{}, fmt.Errorf("new trust during overlap: %w", err)
	}
	degraded := scenarioEvent(id, evidence.Degraded, "lab03-trust-overlap", "trust_rotation", "trust-store", "verification_matrix", "ca_v1=accepted; ca_v2=accepted; overlap=bounded", "drain_old", "a bounded overlap trusts both CA revisions while old sessions drain")
	degraded.DesiredState = "trust-ca-v2-only"
	degraded.EffectiveState = "trust-ca-v1+ca-v2"

	newRoots := certPool(caV2.certificate)
	if err := tlsGet(ctx, serviceV2.URL, "localhost", newRoots, &clientV2); err != nil {
		return evidence.LabResult{}, fmt.Errorf("new trust after rotation: %w", err)
	}
	if err := tlsGet(ctx, serviceV1.URL, "localhost", newRoots, &clientV1); err == nil {
		return evidence.LabResult{}, fmt.Errorf("retired CA unexpectedly trusted after overlap")
	}
	recovery := scenarioEvent(id, evidence.Recovery, "lab03-trust-v2-activation", "trust_rotation", "trust-store", "verification_matrix", "ca_v2=accepted; ca_v1=rejected", "allow", "the new CA becomes the only trusted revision after overlap and new mTLS succeeds")
	recovery.PolicyRevision = "trust-v2"
	recovery.PreconditionRevision = "trust-v1+v2"

	return evidence.NewResult(id, "TLS identity and certificate lifecycle", []evidence.Event{normal, reject, dependency, degraded, recovery})
}
