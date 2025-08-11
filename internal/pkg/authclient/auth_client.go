package authclient

import (
	"context"
	"fmt"
	"log"
	"time"

	authpb "github.com/RehanAthallahAzhar/shopeezy-protos/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type AuthClient struct {
	service authpb.AuthServiceClient
	conn    *grpc.ClientConn
}

func NewAuthClient(grpcServerAddress string) (*AuthClient, error) {
	conn, err := grpc.NewClient(grpcServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("can't connect to gRPC server: %v", err)
	}

	serviceClient := authpb.NewAuthServiceClient(conn)
	return &AuthClient{
		service: serviceClient,
		conn:    conn,
	}, nil
}

/*
	mari kita bahas :

		grpc.WithTransportCredentials(insecure.NewCredentials()),

	mengatur transport credentials (keamanan lapisan transport, seperti TLS/SSL ) yang akan digunakan oleh koneksi gRPC
	insecure.NewCredentials() -> adalah implementasi dari kredensial transportasi yang tidak aman (insecure)
	berarti koneksi gRPC Anda tidak akan menggunakan enkripsi TLS/SSL
	Digunakan terutama untuk pengembangan (development) dan pengujian (testing).
	Dengan lasan kemudahan dan performa (tpi hanya sedikit)

	NAMUN TIDAK BOLEH DIGUNAKAN DI LINGKUNGAN PRODUKSI krn resiko
	- dapat dicegat dan dibaca oleh siapa saja
	- Risiko Man-in-the-Middle (MitM): Penyerang dapat menyusup di antara klien dan server, memodifikasi data, atau berpura-pura menjadi server yang sah.

	Untuk PRODUKSI:
	Menggunakan TLS Anda perlu memiliki sertifikat CA (Certificate Authority) yang dipercaya
	atau sertifikat server yang Anda tahu keasliannya.

		tlsCredentials, err := credentials.NewClientTLSFromFile("path/to/ca.pem", "server.example.com")
*/

func (rc *AuthClient) Close() {
	if rc.conn != nil {
		log.Println("Closing gRPC AuthClient connection...")
		err := rc.conn.Close()
		if err != nil {
			log.Printf("Failed to close gRPC connection: %v", err)
		}
	}
}

func (c *AuthClient) ValidateToken(token string) (isValid bool, userID string, username string, role string, errorMessage string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &authpb.ValidateTokenRequest{Token: token}
	res, err := c.service.ValidateToken(ctx, req)
	if err != nil {
		log.Printf("Gagal memanggil ValidateToken: %v", err)

		// gRPC error handling
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.Unauthenticated {
				return false, "", "", "", st.Message(), st.Err()
			}
		}
		return false, "", "", "", "Kesalahan internal server", err
	}

	if !res.IsValid {
		log.Printf("Token tidak valid: %s", res.GetErrorMessage())
	}

	return res.GetIsValid(), res.GetUserId(), res.GetUsername(), res.GetRole(), res.GetErrorMessage(), nil
}
