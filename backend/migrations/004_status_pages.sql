-- +goose Up

CREATE TABLE status_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    public BOOLEAN NOT NULL DEFAULT false,
    show_uptime BOOLEAN NOT NULL DEFAULT true,
    show_incidents BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_status_pages_slug ON status_pages (slug);

CREATE TABLE status_page_monitors (
    status_page_id UUID NOT NULL REFERENCES status_pages(id) ON DELETE CASCADE,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    display_name TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (status_page_id, monitor_id)
);

CREATE INDEX idx_status_page_monitors_page ON status_page_monitors (status_page_id);
CREATE INDEX idx_status_page_monitors_monitor ON status_page_monitors (monitor_id);

-- +goose Down
DROP INDEX IF EXISTS idx_status_page_monitors_monitor;
DROP INDEX IF EXISTS idx_status_page_monitors_page;
DROP TABLE IF EXISTS status_page_monitors;
DROP INDEX IF EXISTS idx_status_pages_slug;
DROP TABLE IF EXISTS status_pages;
