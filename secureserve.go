package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"crypto/x509/pkix"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Function to generate a password composed of 3 random words
func generatePassword() string {
	words, err := ioutil.ReadFile("/usr/share/dict/words")
	if err != nil {
		fmt.Println("Error reading words file:", err)
		os.Exit(1)
	}
	wordList := strings.Split(string(words), "\n")
	var password string
	for i := 0; i < 3; i++ {
		word := wordList[randInt(0, len(wordList))]
		password += strings.TrimSpace(word)
	}
	return password
}

// Function to generate a random integer
func randInt(min int, max int) int {
	b := make([]byte, 1)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error generating random number:", err)
		os.Exit(1)
	}
	return int(b[0])%(max-min) + min
}

// Function to generate a TLS certificate if it does not already exist
func generateCertificate(certDir string) (string, string) {
	certFile := filepath.Join(certDir, "server.crt")
	keyFile := filepath.Join(certDir, "server.key")

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		createCertificate(certFile, keyFile)
	} else if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		createCertificate(certFile, keyFile)
	}

	return certFile, keyFile
}

// Function to create a new TLS certificate
func createCertificate(certFile string, keyFile string) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Failed to generate private key:", err)
		os.Exit(1)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		fmt.Println("Failed to generate serial number:", err)
		os.Exit(1)
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		Subject:               pkix.Name{CommonName: "localhost"},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		fmt.Println("Failed to create certificate:", err)
		os.Exit(1)
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		fmt.Println("Failed to open cert file for writing:", err)
		os.Exit(1)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.Create(keyFile)
	if err != nil {
		fmt.Println("Failed to open key file for writing:", err)
		os.Exit(1)
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
}

// Function to start a web server
func startWebServer(directory string, certFile string, keyFile string, password string) {
	port := "8081"

	r := mux.NewRouter()
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(directory))))

	// Implement basic auth middleware
	authHandler := handlers.LoggingHandler(os.Stdout, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}
		http.FileServer(http.Dir(directory)).ServeHTTP(w, r)
	}))

	server := &http.Server{
		Addr:      ":" + port,
		TLSConfig: &tls.Config{Certificates: make([]tls.Certificate, 1)},
		Handler:   authHandler,
	}

	server.TLSConfig.Certificates[0], _ = tls.LoadX509KeyPair(certFile, keyFile)

	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			fmt.Println("Server error:", err)
		}
	}()

	// Display URL with all IPs
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Error getting network interfaces:", err)
		os.Exit(1)
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				fmt.Printf("https://%s:%s (Username: user, Password: %s)\n", ipNet.IP.String(), port, password)
			}
		}
	}

	select {}
}

func main() {
	var directory string
	flag.StringVar(&directory, "d", ".", "The directory to serve")
	flag.Parse()

	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		os.Exit(1)
	}

	certDir := filepath.Join(usr.HomeDir, ".local", "share", "secureserve")
	os.MkdirAll(certDir, os.ModePerm)

	password := generatePassword()
	certFile, keyFile := generateCertificate(certDir)
	startWebServer(directory, certFile, keyFile, password)
}

