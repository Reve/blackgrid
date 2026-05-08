package handlers

import (
	"blackgrid/internal/service"
)

type Handlers struct {
	SiteService      *service.SiteService
	VlanService      *service.VlanService
	PrefixService    *service.PrefixService
	IPAddressService *service.IPAddressService
	DeviceService    *service.DeviceService
	DiscoveryService *service.DiscoveryService
}

func New(
	site *service.SiteService,
	vlan *service.VlanService,
	prefix *service.PrefixService,
	ipAddress *service.IPAddressService,
	device *service.DeviceService,
	discovery *service.DiscoveryService,
) *Handlers {
	return &Handlers{
		SiteService:      site,
		VlanService:      vlan,
		PrefixService:    prefix,
		IPAddressService: ipAddress,
		DeviceService:    device,
		DiscoveryService: discovery,
	}
}
