-- +goose Up
CREATE TABLE ping (
    id SERIAL PRIMARY KEY
);

-- +goose Down
DROP TABLE ping;

