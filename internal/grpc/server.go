package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	apperrors "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/services"

	productpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/product"
)

type ProductServer struct {
	productpb.UnimplementedProductServiceServer
	ProductSvc services.ProductService
}

func NewProductServer(productSvc services.ProductService) *ProductServer {
	return &ProductServer{
		ProductSvc: productSvc,
	}
}

func (s *ProductServer) GetProducts(ctx context.Context, req *productpb.GetProductsRequest) (*productpb.GetProductsResponse, error) {
	ids := make([]uuid.UUID, 0, len(req.GetIds()))

	for _, idStr := range req.GetIds() {
		parsedID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "format Product ID '%s' tidak valid", idStr)
		}
		ids = append(ids, parsedID)
	}

	dbProducts, err := s.ProductSvc.GetProductByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	var protoProducts []*productpb.Product
	for _, p := range dbProducts {
		protoProducts = append(protoProducts, &productpb.Product{
			Id:        p.ID.String(),
			SellerId:  p.SellerID.String(),
			Name:      p.Name,
			Price:     int32(p.Price),
			Stock:     int32(p.Stock),
			CreatedAt: timestamppb.New(p.CreatedAt),
			UpdatedAt: timestamppb.New(p.UpdatedAt),
		})
	}

	return &productpb.GetProductsResponse{Products: protoProducts}, nil
}

func (s *ProductServer) DecreaseStock(ctx context.Context, req *productpb.DecreaseStockRequest) (*productpb.DecreaseStockResponse, error) {
	updatedProducts, err := s.ProductSvc.DecreaseStock(ctx, req.GetItems())
	if err != nil {
		if errors.Is(err, apperrors.ErrProductOutOfStock) {
			return nil, status.Errorf(codes.FailedPrecondition, "product out of stock: %v", err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to decrease stock: %v", err)
	}

	pbProducts := make([]*productpb.Product, len(updatedProducts))
	for i, p := range updatedProducts {
		pbProducts[i] = &productpb.Product{
			Id:          p.ID.String(),
			SellerId:    p.SellerID.String(),
			Name:        p.Name,
			Price:       int32(p.Price),
			Stock:       int32(p.Stock),
			Description: p.Description,
			CreatedAt:   timestamppb.New(p.CreatedAt),
			UpdatedAt:   timestamppb.New(p.UpdatedAt),
		}
	}

	return &productpb.DecreaseStockResponse{
		Products: pbProducts,
	}, nil
}

func (s *ProductServer) IncreaseStock(ctx context.Context, req *productpb.IncreaseStockRequest) (*productpb.IncreaseStockResponse, error) {
	updatedProducts, err := s.ProductSvc.IncreaseStock(ctx, req.GetItems())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to increase stock: %v", err)
	}

	pbProducts := make([]*productpb.Product, len(updatedProducts))
	for i, p := range updatedProducts {
		pbProducts[i] = &productpb.Product{
			Id:          p.ID.String(),
			SellerId:    p.SellerID.String(),
			Name:        p.Name,
			Price:       int32(p.Price),
			Stock:       int32(p.Stock),
			Description: p.Description,
			CreatedAt:   timestamppb.New(p.CreatedAt),
			UpdatedAt:   timestamppb.New(p.UpdatedAt),
		}
	}

	return &productpb.IncreaseStockResponse{
		Products: pbProducts,
	}, nil
}
