//go:generate npm run export --silent --prefix ../ui
//go:generate go-bindata -pkg server -nocompress -o ./bindata_ui.go -fs -prefix "../ui/out/" ../ui/out/...

package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/generate"
	"golang.org/x/crypto/acme/autocert"
	"gorm.io/gorm"
)

type StaticFileSystem struct {
	base http.FileSystem
}

func (sfs StaticFileSystem) Open(name string) (http.File, error) {
	f, err := sfs.base.Open(name)
	if os.IsNotExist(err) {
		if f, err := sfs.base.Open(name + ".html"); err == nil {
			return f, nil
		}
		return sfs.base.Open("index.html")
	}

	if err != nil {
		return nil, err
	}

	return f, nil
}

type Sync struct {
	stop chan bool
}

func generateSelfSignedCert(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			CommonName:   generate.RandString(25),
			Organization: []string{generate.RandString(25)},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 1, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			CommonName:   generate.RandString(25),
			Organization: []string{generate.RandString(25)},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 1, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	if hello.ServerName != "" {
		cert.DNSNames = []string{hello.ServerName}
	} else {
		tcp, ok := hello.Conn.LocalAddr().(*net.TCPAddr)
		if !ok {
			return nil, errors.New("could not determine ip")
		}
		cert.IPAddresses = append(cert.IPAddresses, tcp.IP)
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	keypair, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, err
	}

	return &keypair, nil
}

func getSelfSignedOrLetsEncryptCert(certManager *autocert.Manager) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	selfSignCache := make(map[string]*tls.Certificate)

	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {

		cert, err := certManager.GetCertificate(hello)
		if err == nil {
			return cert, nil
		}

		name := hello.ServerName
		if name == "" {
			name = hello.Conn.LocalAddr().String()
		}

		cert, ok := selfSignCache[name]
		if !ok {
			cert, err = generateSelfSignedCert(hello)
			if err != nil {
				return nil, err
			}
			selfSignCache[name] = cert
		}

		return cert, nil
	}
}

const SYNC_INTERVAL_SECONDS = 10

func (s *Sync) Start(sync func()) {
	ticker := time.NewTicker(SYNC_INTERVAL_SECONDS * time.Second)
	sync()

	go func() {
		for {
			select {
			case <-ticker.C:
				sync()
			case <-s.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Sync) Stop() {
	s.stop <- true
}

type ServerOptions struct {
	DBPath     string
	ConfigPath string
	TLSCache   string
	UI         bool
	UIProxy    bool
}

func Run(options *ServerOptions) error {
	if options.DBPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		options.DBPath = filepath.Join(homeDir, ".infra")
	}

	db, err := NewDB(options.DBPath)
	if err != nil {
		return err
	}

	var settings Settings
	err = db.First(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		settings.TokenSecret = []byte(generate.RandString(32))
		db.Save(&settings)
	}

	kube, err := NewKubernetes()
	if err != nil {
		fmt.Println("warning: could connect to Kubernetes API", err)
	}

	fmt.Println(options)

	var config Config
	err = LoadConfig(&config, options.ConfigPath)
	if err != nil {
		fmt.Println("warning: could not open config file: ", err)
	}

	sync := Sync{}
	sync.Start(func() {
		if config.Providers.Okta.Domain != "" {
			emails, err := config.Providers.Okta.Emails()
			if err != nil {
				fmt.Println(err)
			}

			err = SyncUsers(db, emails, "okta")
			if err != nil {
				fmt.Println(err)
			}
			kube.UpdatePermissions(db, &config)
		}
	})

	defer sync.Stop()

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()
	unixRouter.Use(gin.Logger())
	unixRouter.Use(func(c *gin.Context) {
		c.Set("skipauth", true)
	})

	if err = addRoutes(unixRouter, db, kube, &config, &settings); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	os.Remove(filepath.Join(homeDir, ".infra", "infra.sock"))
	go func() {
		if err := unixRouter.RunUnix(filepath.Join(homeDir, ".infra", "infra.sock")); err != nil {
			log.Fatal(err)
		}
	}()

	router := gin.New()
	router.Use(gin.Logger())
	if err = addRoutes(router, db, kube, &config, &settings); err != nil {
		return err
	}

	if options.UIProxy {
		remote, _ := url.Parse("http://localhost:3000")
		devProxy := httputil.NewSingleHostReverseProxy(remote)
		router.NoRoute(func(c *gin.Context) {
			devProxy.ServeHTTP(c.Writer, c.Request)
		})
	} else if options.UI {
		router.NoRoute(func(c *gin.Context) {
			gziphandler.GzipHandler(http.FileServer(&StaticFileSystem{base: AssetFile()})).ServeHTTP(c.Writer, c.Request)
		})
	}

	go router.Run(":80")

	if err := os.MkdirAll(options.TLSCache, os.ModePerm); err != nil {
		return err
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(options.TLSCache),
	}

	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = getSelfSignedOrLetsEncryptCert(manager)

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   router,
	}

	return tlsServer.ListenAndServeTLS("", "")
}
