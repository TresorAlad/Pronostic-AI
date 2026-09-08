-- model_version was VARCHAR(50); ML bundle ids can exceed that limit.
ALTER TABLE predictions ALTER COLUMN model_version TYPE VARCHAR(255);
