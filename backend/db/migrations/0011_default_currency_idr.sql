-- +goose Up
UPDATE chart_of_accounts SET currency = 'IDR' WHERE currency = 'USD';
UPDATE product SET currency = 'IDR' WHERE currency = 'USD';

-- +goose Down
UPDATE chart_of_accounts SET currency = 'USD' WHERE currency = 'IDR';
UPDATE product SET currency = 'USD' WHERE currency = 'IDR';
