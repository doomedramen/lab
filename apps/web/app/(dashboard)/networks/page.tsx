"use client";

import { Network } from "lucide-react";
import { PageHeader } from "@/components/page-header";
import { NetworkList } from "@/components/network-list";
import { useNetworks } from "@/lib/api/queries";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function NetworksPage() {
  const { data, isLoading } = useNetworks();

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/"
        backLabel="Dashboard"
        title="Networks"
        subtitle="Manage virtual networks and firewall rules"
        icon={<Network className="size-5 text-foreground" />}
      />

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Total Networks
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data?.total ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Active Networks
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {data?.networks?.filter((n) => n.status === 1).length ?? 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Total Interfaces
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {data?.networks?.reduce(
                (sum, n) => sum + (n.interfaceCount ?? 0),
                0,
              ) ?? 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">DHCP Networks</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {data?.networks?.filter((n) => n.dhcpEnabled).length ?? 0}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Virtual Networks</CardTitle>
        </CardHeader>
        <CardContent>
          <NetworkList networks={data?.networks} isLoading={isLoading} />
        </CardContent>
      </Card>
    </div>
  );
}
