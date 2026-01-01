package account

import (
	"context"
	"fmt"

	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AccountClient struct {
	Service accountpb.AccountServiceClient
	Conn    *grpc.ClientConn
}

func NewAccountClient(grpcServerAddress string) (*AccountClient, error) {
	conn, err := grpc.NewClient(grpcServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("can't connect to gRPC server: %v", err)
	}

	serviceClient := accountpb.NewAccountServiceClient(conn)

	return &AccountClient{
		Service: serviceClient,
		Conn:    conn,
	}, nil
}

func (c *AccountClient) Close() {
	c.Conn.Close()
}

// GetUser calls the GetUser RPC with just the ID
func (c *AccountClient) GetUser(ctx context.Context, id string) (*accountpb.User, error) {
	req := &accountpb.GetUserRequest{
		Id: id,
	}
	return c.Service.GetUser(ctx, req)
}

// GetUsers calls the GetUsers RPC with a list of IDs
func (c *AccountClient) GetUsers(ctx context.Context, ids []string) (*accountpb.GetUsersResponse, error) {
	req := &accountpb.GetUsersRequest{
		Ids: ids,
	}
	return c.Service.GetUsers(ctx, req)
}
