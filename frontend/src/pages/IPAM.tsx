import { useEffect, useState } from 'react';
import { getPrefixes, getIPAddresses } from '../api/client';
import type { Prefix, IPAddress } from '../api/client';

export default function IPAM() {
  const [prefixes, setPrefixes] = useState<Prefix[]>([]);
  const [ips, setIps] = useState<IPAddress[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [prefixesRes, ipsRes] = await Promise.all([
          getPrefixes(),
          getIPAddresses()
        ]);
        // Handle null responses from empty DB
        setPrefixes(prefixesRes.data || []);
        setIps(ipsRes.data || []);
      } catch (error) {
        console.error("Failed to fetch IPAM data", error);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="p-4 text-signal-amber">Loading IPAM data...</div>;

  return (
    <div className="space-y-6">
      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">Prefixes</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">Prefix</th>
                <th className="p-2">Description</th>
              </tr>
            </thead>
            <tbody>
              {prefixes.length === 0 ? (
                <tr><td colSpan={2} className="p-2 text-text-muted">No prefixes found</td></tr>
              ) : prefixes.map(p => (
                <tr key={p.id} className="border-b border-surface">
                  <td className="p-2 text-signal-green">{p.prefix}</td>
                  <td className="p-2 text-text-muted">{p.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">IP Addresses</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">IP Address</th>
                <th className="p-2">Status</th>
                <th className="p-2">Description</th>
              </tr>
            </thead>
            <tbody>
              {ips.length === 0 ? (
                <tr><td colSpan={3} className="p-2 text-text-muted">No IP addresses found</td></tr>
              ) : ips.map(ip => (
                <tr key={ip.id} className="border-b border-surface">
                  <td className="p-2 text-signal-green">{ip.ip_address}</td>
                  <td className="p-2">
                    <span className={`px-2 py-1 rounded text-xs ${ip.status === 'active' ? 'bg-signal-green/20 text-signal-green' : 'bg-surface text-text-muted'}`}>
                      {ip.status}
                    </span>
                  </td>
                  <td className="p-2 text-text-muted">{ip.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
