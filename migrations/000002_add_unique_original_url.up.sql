CREATE UNIQUE INDEX IF NOT EXISTS short_urls_original_url_uq
    ON short_urls (original_url);