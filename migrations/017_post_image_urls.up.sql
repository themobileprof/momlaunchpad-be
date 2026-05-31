-- External image URLs on community posts (HTTPS links only; no file storage).
ALTER TABLE community_posts
    ADD COLUMN IF NOT EXISTS image_urls TEXT[] NOT NULL DEFAULT '{}';
