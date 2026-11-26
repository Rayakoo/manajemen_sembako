-- Add total_price (total batch purchase price) to stock_batches.
-- buy_price stores the derived unit buy price (total_price / initial_stock).

ALTER TABLE stock_batches
    ADD COLUMN IF NOT EXISTS total_price NUMERIC(15, 2);

UPDATE stock_batches
SET total_price = ROUND(buy_price * initial_stock, 2)
WHERE total_price IS NULL;

ALTER TABLE stock_batches
    ALTER COLUMN total_price SET NOT NULL,
    ALTER COLUMN total_price SET DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_stock_batches_total_price_non_negative'
    ) THEN
        ALTER TABLE stock_batches
            ADD CONSTRAINT chk_stock_batches_total_price_non_negative CHECK (total_price >= 0);
    END IF;
END $$;
