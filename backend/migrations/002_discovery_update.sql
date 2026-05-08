-- +goose Up

-- Update prefixes table
ALTER TABLE prefixes
ADD COLUMN scan_enabled BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN scan_interval_seconds INTEGER NOT NULL DEFAULT 3600;

-- Update ip_addresses table
ALTER TABLE ip_addresses
ADD COLUMN last_seen_at TIMESTAMP WITH TIME ZONE;

-- Drop existing discovery tables so we can recreate them with the correct fields
DROP TABLE IF EXISTS discovery_results;
DROP TABLE IF EXISTS discovery_scans;

CREATE TABLE discovery_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prefix_id UUID REFERENCES prefixes(id) ON DELETE CASCADE NOT NULL,
    status VARCHAR(50) NOT NULL, -- queued, running, completed, failed, cancelled
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE discovery_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID REFERENCES discovery_scans(id) ON DELETE CASCADE NOT NULL,
    prefix_id UUID REFERENCES prefixes(id) ON DELETE CASCADE NOT NULL,
    address INET NOT NULL,
    mac_address MACADDR,
    hostname TEXT,
    reverse_dns TEXT,
    open_ports JSONB NOT NULL DEFAULT '[]',
    latency_ms INTEGER,
    classification TEXT NOT NULL,
    seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    raw JSONB NOT NULL DEFAULT '{}',
    ignored BOOLEAN NOT NULL DEFAULT false,
    accepted_at TIMESTAMP WITH TIME ZONE,
    created_ip_address_id UUID REFERENCES ip_addresses(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE discovery_results;
DROP TABLE discovery_scans;

ALTER TABLE ip_addresses DROP COLUMN last_seen_at;
ALTER TABLE prefixes DROP COLUMN scan_interval_seconds, DROP COLUMN scan_enabled;
