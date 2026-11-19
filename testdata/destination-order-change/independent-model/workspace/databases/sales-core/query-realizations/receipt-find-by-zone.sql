SELECT lot_code, receipt_seq, units
FROM public.stock_receipts
WHERE zone_code = :zone_code
ORDER BY lot_code, receipt_seq;
