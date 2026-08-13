-- +goose Up
INSERT INTO chart_of_accounts (code, name, type, currency) VALUES
  ('1000', 'Cash at Central Bank', 'asset',     'USD'),
  ('1100', 'Loans Receivable',     'asset',     'USD'),
  ('2000', 'Customer Deposits',    'liability', 'USD'),
  ('3000', 'Retained Earnings',    'equity',    'USD'),
  ('4000', 'Interest Income',      'income',    'USD'),
  ('4100', 'Fee Income',           'income',    'USD'),
  ('5000', 'Interest Expense',     'expense',   'USD');

-- +goose Down
DELETE FROM chart_of_accounts
WHERE code IN ('1000', '1100', '2000', '3000', '4000', '4100', '5000');
