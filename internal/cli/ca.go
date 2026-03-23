package cli

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/mischa/codingbox/internal/proxy"
	"github.com/spf13/cobra"
)

var caCmd = &cobra.Command{
	Use:   "ca",
	Short: "Manage the CA certificate",
	Long:  "Manage the CA certificate used for TLS interception.",
}

var caShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show CA certificate info",
	RunE:  runCAShow,
}

var caRegenerateCmd = &cobra.Command{
	Use:   "regenerate",
	Short: "Regenerate the CA certificate",
	RunE:  runCARegenerate,
}

func init() {
	caCmd.AddCommand(caShowCmd)
	caCmd.AddCommand(caRegenerateCmd)
	rootCmd.AddCommand(caCmd)
}

func runCAShow(cmd *cobra.Command, args []string) error {
	cm := proxy.NewCertManager(proxy.ConfigDir())
	certPath := cm.CertPath()

	data, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("CA certificate not found at %s\nRun 'codingbox run' to generate one, or 'codingbox ca regenerate'", certPath)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return fmt.Errorf("invalid PEM data in %s", certPath)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parsing certificate: %w", err)
	}

	fingerprint := sha256.Sum256(cert.Raw)
	fmt.Printf("Path:        %s\n", certPath)
	fmt.Printf("Subject:     %s\n", cert.Subject.CommonName)
	fmt.Printf("Not Before:  %s\n", cert.NotBefore.Format("2006-01-02"))
	fmt.Printf("Not After:   %s\n", cert.NotAfter.Format("2006-01-02"))
	fmt.Printf("Fingerprint: %x\n", fingerprint)
	return nil
}

func runCARegenerate(cmd *cobra.Command, args []string) error {
	cm := proxy.NewCertManager(proxy.ConfigDir())
	_, err := cm.Regenerate()
	if err != nil {
		return err
	}
	fmt.Printf("CA certificate regenerated at %s\n", cm.CertPath())
	return nil
}
