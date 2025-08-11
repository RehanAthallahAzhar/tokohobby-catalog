-- name: CreateOrderItem :exec
INSERT INTO order_items (id, order_id, product_id, quantity, price)
VALUES ($1, $2, $3, $4, $5);

-- name: GetOrderItemByOrderID :many
SELECT * FROM order_items WHERE order_id = $1;

-- name: GetOrderItemByProductID :many
SELECT * FROM order_items WHERE product_id = $1;