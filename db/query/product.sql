-- name: GetAllProducts :many
SELECT 
  products.id,
  products.seller_id,
  users.name as seller_name,
  products.name,
  products.price,
  products.stock,
  products.discount,
  products.type,
  products.description,
  products.created_at,
  products.updated_at
FROM products
LEFT JOIN users ON users.id = products.seller_id
WHERE products.deleted_at IS NULL;

-- name: GetProductByID :one
SELECT 
  products.id,
  products.seller_id,
  users.name as seller_name,
  products.name,
  products.price,
  products.stock,
  products.discount,
  products.type,
  products.description,
  products.created_at,
  products.updated_at
FROM products
LEFT JOIN users ON users.id = products.seller_id
WHERE products.id = $1 AND products.deleted_at IS NULL;

-- name: GetProductsBySellerID :many
SELECT 
  products.id,
  products.seller_id,
  users.name as seller_name,
  products.name,
  products.price,
  products.stock,
  products.discount,
  products.type,
  products.description,
  products.created_at,
  products.updated_at
FROM products
LEFT JOIN users ON users.id = products.seller_id
WHERE products.seller_id = $1 AND products.deleted_at IS NULL;

-- name: GetProductsByName :many
SELECT 
  products.id,
  products.seller_id,
  users.name as seller_name,
  products.name,
  products.price,
  products.stock,
  products.discount,
  products.type,
  products.description,
  products.created_at,
  products.updated_at
FROM products
LEFT JOIN users ON users.id = products.seller_id
WHERE products.name LIKE $1 AND products.deleted_at IS NULL;

-- name: InsertProduct :one
INSERT INTO products (
  id, seller_id, name, price, stock, discount, type, description, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
)
RETURNING *;

-- name: UpdateProduct :exec
UPDATE products
SET name = $2, price = $3, stock = $4, discount = $5, type = $6, description = $7, updated_at = NOW()
WHERE id = $1 AND seller_id = $8;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1;

-- name: GetProductStock :one
SELECT stock FROM products WHERE id = $1;

-- name: UpdateProductStock :exec
UPDATE products SET stock = $2 WHERE id = $1;
