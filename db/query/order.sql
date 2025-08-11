-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1;

-- name: GetOrderByUserID :many
SELECT * FROM orders WHERE user_id = $1;

-- name: GetOrderByStatus :many
SELECT * FROM orders WHERE status = $1;

-- name: GetOrderByDate :many
SELECT * FROM orders WHERE order_date = $1;

-- name: CreateOrder :exec
INSERT INTO orders (id, user_id, status, order_date, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateOrderTotalAmount :exec
UPDATE orders
SET total_amount = $1, updated_at = $2
WHERE id = $3;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = $1, updated_at = $2
WHERE id = $3;

-- name: DeleteOrder :exec
DELETE FROM orders WHERE id = $1;


