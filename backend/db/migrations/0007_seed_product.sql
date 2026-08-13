-- +goose Up
INSERT INTO product (code, name, kind, currency, interest_rate_bps)
VALUES ('SAV', 'Basic Savings', 'deposit', 'USD', 150);

-- +goose Down
DELETE FROM product WHERE code = 'SAV';
