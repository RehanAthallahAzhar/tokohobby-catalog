-- name: GetCartItem :one
SELECT * FROM carts WHERE user_id = $1 AND product_id = $2;

-- name: GetCartItems :many
SELECT * FROM carts WHERE user_id = $1;

-- name: InsertCartItem :exec
INSERT INTO carts (id, user_id, product_id, quantity, "description")
VALUES ($1, $2, $3, $4, $5);

-- name: UpdateCartItem :exec
UPDATE carts
SET quantity = $1, "description" = $2
WHERE user_id = $3 AND product_id = $4;

-- name: DeleteCart :exec
DELETE FROM carts WHERE id = $1 AND product_id = $2;


