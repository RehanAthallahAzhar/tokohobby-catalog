package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/db"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/errors"
)

type ProductRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateProduct(ctx context.Context, product *db.InsertProductParams) (*db.Product, error)
	GetAllProducts(ctx context.Context) ([]db.GetAllProductsRow, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (*db.GetProductByIDRow, error)
	GetProductByIDs(ctx context.Context, ids []uuid.UUID) ([]db.GetProductByIDsRow, error)
	GetProductsBySellerID(ctx context.Context, sellerID uuid.UUID) ([]db.GetProductsBySellerIDRow, error)
	GetProductsByName(ctx context.Context, name string) ([]db.GetProductsByNameRow, error)
	GetProductsByType(ctx context.Context, productType string) ([]db.GetProductsByTypeRow, error)
	UpdateProduct(ctx context.Context, updateParams *db.UpdateProductParams) (*db.Product, error)
	DeleteProduct(ctx context.Context, id uuid.UUID) (*db.Product, error)
	DecreaseProductStock(ctx context.Context, tx *sql.Tx, productID uuid.UUID, quantity int32) (*db.Product, error)
	IncreaseProductStock(ctx context.Context, tx *sql.Tx, params db.IncreaseProductStockParams) (db.Product, error)
}

type productRepository struct {
	db  *sql.DB
	q   *db.Queries
	log *logrus.Logger
}

func NewProductRepository(
	db *sql.DB,
	q *db.Queries,
	log *logrus.Logger,
) ProductRepository {
	return &productRepository{
		db:  db,
		q:   q,
		log: log,
	}
}

func (r *productRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *productRepository) CreateProduct(ctx context.Context, product *db.InsertProductParams) (*db.Product, error) {
	row, err := r.q.InsertProduct(ctx, *product)
	if err != nil {
		r.log.WithField("product_id", product.ID).WithError(err).Error("Failed to create product in the database")
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return &row, err
}

func (r *productRepository) GetAllProducts(ctx context.Context) ([]db.GetAllProductsRow, error) {
	var rows []db.GetAllProductsRow

	rows, err := r.q.GetAllProducts(ctx)
	if err != nil {
		r.log.WithFields(logrus.Fields{"error": err}).Error("Failed to receive orders from DB")
		return nil, err
	}

	return rows, nil
}

func (r *productRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*db.GetProductByIDRow, error) {
	var row db.GetProductByIDRow

	row, err := r.q.GetProductByID(ctx, id)
	if err != nil {
		r.log.WithFields(logrus.Fields{"id": id, "error": err}).Error("Failed to receive product from DB")
		return nil, fmt.Errorf("failed to receive product from DB: %w", err)
	}

	return &row, nil
}

func (r *productRepository) GetProductByIDs(ctx context.Context, ids []uuid.UUID) ([]db.GetProductByIDsRow, error) {
	var rows []db.GetProductByIDsRow

	rows, err := r.q.GetProductByIDs(ctx, ids)
	if err != nil {
		r.log.WithFields(logrus.Fields{"product_ids": ids, "error": err}).Error("Failed to receive product from DB")
		return nil, err
	}

	return rows, nil
}

func (r *productRepository) GetProductsBySellerID(ctx context.Context, sellerID uuid.UUID) ([]db.GetProductsBySellerIDRow, error) {
	var rows []db.GetProductsBySellerIDRow

	rows, err := r.q.GetProductsBySellerID(ctx, sellerID)
	if err != nil {
		r.log.WithFields(logrus.Fields{"seller_id": sellerID}).WithError(err).Error("Failed to receive products by seller ID from DB")
		return nil, err
	}

	return rows, nil
}

func (r *productRepository) GetProductsByName(ctx context.Context, name string) ([]db.GetProductsByNameRow, error) {
	var rows []db.GetProductsByNameRow

	searchPattern := "%" + name + "%"

	rows, err := r.q.GetProductsByName(ctx, searchPattern)
	if err != nil {
		r.log.WithFields(logrus.Fields{
			"name_query": name,
			"error":      err,
		}).Error("Failed to execute the GetProductsByName query in the database")
		return nil, err
	}

	r.log.WithFields(logrus.Fields{"name": name, "error": err}).Error("Product not found")
	return rows, nil
}

func (r *productRepository) GetProductsByType(ctx context.Context, productType string) ([]db.GetProductsByTypeRow, error) {
	var rows []db.GetProductsByTypeRow

	rows, err := r.q.GetProductsByType(ctx, sql.NullString{
		String: productType,
		Valid:  true,
	})

	if err != nil {
		r.log.WithFields(logrus.Fields{
			"name_query": productType,
			"error":      err,
		}).Error("Failed to execute the GetProductsByName query in the database")
		return nil, err
	}

	r.log.WithFields(logrus.Fields{"name": productType, "error": err}).Error("Product not found")
	return rows, nil
}

func (r *productRepository) UpdateProduct(ctx context.Context, updateParams *db.UpdateProductParams) (*db.Product, error) {
	var row db.Product

	row, err := r.q.UpdateProduct(ctx, *updateParams)

	if err != nil {
		r.log.WithField("product_id", updateParams.ID).WithError(err).Error("Failed to update product in the database")
		return nil, err
	}

	return &row, nil
}

func (r *productRepository) DeleteProduct(ctx context.Context, id uuid.UUID) (*db.Product, error) {
	var row db.Product

	row, err := r.q.DeleteProduct(ctx, id)
	if err != nil {
		r.log.WithField("product_id", id).WithError(err).Error("Failed to delete product in the database")
		return nil, err
	}

	return &row, nil
}

func (r *productRepository) DecreaseProductStock(ctx context.Context, tx *sql.Tx, productID uuid.UUID, quantity int32) (*db.Product, error) {
	q := r.q.WithTx(tx)

	updatedProduct, err := q.DecreaseProductStock(ctx, db.DecreaseProductStockParams{
		ProductID: productID,
		Quantity:  quantity,
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrProductOutOfStock
		}
		return nil, fmt.Errorf("failed to decrease stock: %w", err)
	}

	return &updatedProduct, nil
}

func (r *productRepository) IncreaseProductStock(ctx context.Context, tx *sql.Tx, params db.IncreaseProductStockParams) (db.Product, error) {
	qtx := r.q.WithTx(tx)
	updatedProduct, err := qtx.IncreaseProductStock(ctx, params)
	if err != nil {
		return db.Product{}, fmt.Errorf("failed to increase stock: %w", err)
	}
	return updatedProduct, nil
}
