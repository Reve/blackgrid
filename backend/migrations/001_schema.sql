-- +goose Up
CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE vlans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID REFERENCES sites(id) ON DELETE CASCADE,
    vlan_id INTEGER NOT NULL CHECK (vlan_id >= 1 AND vlan_id <= 4094),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(site_id, vlan_id)
);

CREATE TABLE prefixes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID REFERENCES sites(id) ON DELETE CASCADE,
    vlan_id UUID REFERENCES vlans(id) ON DELETE SET NULL,
    prefix VARCHAR(45) NOT NULL, -- CIDR notation
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(prefix)
);

CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE interfaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID REFERENCES devices(id) ON DELETE CASCADE NOT NULL,
    name VARCHAR(255) NOT NULL,
    mac_address VARCHAR(17),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, name)
);

CREATE TABLE ip_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prefix_id UUID REFERENCES prefixes(id) ON DELETE CASCADE NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    interface_id UUID REFERENCES interfaces(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'active',
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(ip_address)
);

CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    target VARCHAR(255) NOT NULL,
    interval_seconds INTEGER DEFAULT 60,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE monitor_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID REFERENCES monitors(id) ON DELETE CASCADE NOT NULL,
    status VARCHAR(50) NOT NULL,
    latency_ms INTEGER,
    error_message TEXT,
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID REFERENCES monitors(id) ON DELETE CASCADE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE,
    description TEXT
);

CREATE TABLE discovery_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prefix_id UUID REFERENCES prefixes(id) ON DELETE CASCADE NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE discovery_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID REFERENCES discovery_scans(id) ON DELETE CASCADE NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    mac_address VARCHAR(17),
    hostname VARCHAR(255),
    is_active BOOLEAN DEFAULT false,
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action VARCHAR(255) NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    entity_id UUID NOT NULL,
    changes JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
