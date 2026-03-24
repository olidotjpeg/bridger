CREATE INDEX IF NOT EXISTS idx_images_rating       ON images(rating);
CREATE INDEX IF NOT EXISTS idx_images_capture_date ON images(capture_date);
CREATE INDEX IF NOT EXISTS idx_image_tags_tag_id   ON image_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_image_tags_image_id ON image_tags(image_id);