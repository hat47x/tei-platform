SELECT
    r.request_id,
    r.summary,
    r.status
FROM public.requests AS r
ORDER BY r.request_id
