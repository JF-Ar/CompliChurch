ALTER TABLE items
  DROP CONSTRAINT IF EXISTS items_status_check;

ALTER TABLE items
  ADD CONSTRAINT items_status_check
  CHECK (status IN ('available', 'on_loan', 'maintenance', 'damaged'));
