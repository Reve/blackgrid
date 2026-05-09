sed -i "s/import React from 'react';//" frontend/src/pages/Dashboard.tsx
sed -i "s/import React, { useEffect, useState } from 'react';/import { useEffect, useState } from 'react';/" frontend/src/pages/Devices.tsx
sed -i "s/import { getDevices, Device } from '..\/api\/client';/import { getDevices } from '..\/api\/client';\nimport type { Device } from '..\/api\/client';/" frontend/src/pages/Devices.tsx
sed -i "s/import React, { useEffect, useState } from 'react';/import { useEffect, useState } from 'react';/" frontend/src/pages/IPAM.tsx
sed -i "s/import { getPrefixes, getIPAddresses, Prefix, IPAddress } from '..\/api\/client';/import { getPrefixes, getIPAddresses } from '..\/api\/client';\nimport type { Prefix, IPAddress } from '..\/api\/client';/" frontend/src/pages/IPAM.tsx
sed -i "s/import React from 'react';//" frontend/src/pages/Incidents.tsx
sed -i "s/import React from 'react';//" frontend/src/pages/Monitors.tsx
sed -i "s/import React from 'react';//" frontend/src/pages/Settings.tsx
