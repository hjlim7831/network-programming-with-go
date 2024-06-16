package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

var (
	// 호스트 네임과 IP 주소를 쉼표로 구분된 문자열로 받음
	host = flag.String("host", "localhost",
		"Certificate's comma-separated host names and IPs")
	certFn = flag.String("cert", "cert.pem", "certificate file name")
	keyFn  = flag.String("key", "key.pem", "private key file name")
)

func main() {
	flag.Parse()
	// 직접 서명한 인증서를 생성하므로, 암호학적으로 랜덤하고 부호가 없는 128비트의 정수를 사용해 일련의 번호를 생성
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		log.Fatal(err)
	}

	notBefore := time.Now()
	// X.509 포맷으로 인코딩된 인증서를 나타내는 x509.Certificate 객체를 생성
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Adam Woodbeck"},
		},
		NotBefore: notBefore,
		NotAfter:  notBefore.Add(10 * 356 * 24 * time.Hour),
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			// 이 인증서를 클라이언트와의 인증에서 사용할 것이므로, 이 값을 반드시 포함해야 함
			// 이 값을 포함하지 않으면, 클라이언트가 TLS 협상 단계에서 해당 인증서를 사용할 때 서버에서는 클라이언트를 확인할 수 없음
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// 호스트 네임과 IP 주소를 쉼표로 구분된 문자열로 받아들임
	// 쉼표를 기준으로 나눈 뒤, 각 값을 해당하는 템플릿 내의 슬라이스에 할당
	// 이 값들은 클라이언트 인증서의 CN 값이나 SAN 값을 얻는 데에 사용됨
	// Go의 TLS 클라이언트는 이 값들을 이용해 서버와 인증하지만, 서버는 클라이언트를 인증할 때 클라이어트의 인증서로부터 이 값들을 이용하지 않음
	for _, h := range strings.Split(*host, ",") {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// P-256 타원 곡선을 이용해 새로운 ECDSA 개인키를 생성
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	// 암호화를 위해 무작위 값을 추출하기 위한 엔트로피 소스
	// 새로운 인증서를 생성하기 위한 템플릿, 상위 인증서, 공개키, 개인키를 매개변수로 받음
	// DER(Distinguished Encoding Rules) 포맷으로 인코딩된 인증서가 포함된 바이트 슬라이스를 반환
	// 스스로 서명한 인증서를 사용하기 때문에, 상위 인증서의 템플릿을 사용
	der, err := x509.CreateCertificate(rand.Reader, &template,
		&template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := os.Create(*certFn)
	if err != nil {
		log.Fatal(err)
	}

	// DER 포맷으로 인코딩된 바이트 슬라이스의 pem.Block 객체를 생성한 뒤, 모든 데이터를 PEM 포맷으로 인코딩해 새로운 파일에 저장
	err = pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err != nil {
		log.Fatal(err)
	}

	if err := cert.Close(); err != nil {
		log.Fatal(err)
	}
	log.Println("wrote", *certFn)

	// 프로그램을 실행하는 사용자에게만 개이키 파일에 읽기-쓰기 권한을 줌
	key, err := os.OpenFile(*keyFn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatal(err)
	}

	// 개인키를 바이트 슬라이스로 마샬링
	privKey, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatal(err)
	}

	// 개인키 파일을 PEM 포맷으로 인코딩된 파일로 쓰기 전에, 먼저 마샬링된 바이트 슬라이스를 pem.Block 객체에 할당
	err = pem.Encode(key, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privKey})
	if err != nil {
		log.Fatal(err)
	}

	if err := key.Close(); err != nil {
		log.Fatal(err)
	}
	log.Println("wrote", *keyFn)
}
