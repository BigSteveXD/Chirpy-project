-- name: UpdateUser :exec
UPDATE users
SET updated_at=$1, email=$2, hashed_password=$3 WHERE id=$4;
