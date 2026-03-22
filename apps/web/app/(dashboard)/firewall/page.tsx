"use client";

import { Shield, Activity, CheckCircle2, XCircle } from "lucide-react";
import { PageHeader } from "@/components/page-header";
import { FirewallRulesList } from "@/components/firewall-rules-list";
import { useFirewallRules, useFirewallStatus } from "@/lib/api/queries";
import { useFirewallMutations } from "@/lib/api/mutations/network";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

export default function FirewallPage() {
  const { data: rulesData, isLoading } = useFirewallRules();
  const { data: status } = useFirewallStatus();
  const { enableFirewall, disableFirewall } = useFirewallMutations();

  const enabledRules = rulesData?.rules?.filter((r) => r.enabled).length ?? 0;
  const disabledRules = rulesData?.rules?.filter((r) => !r.enabled).length ?? 0;

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/"
        backLabel="Dashboard"
        title="Firewall"
        subtitle="Manage firewall rules and network security"
        icon={<Shield className="size-5 text-foreground" />}
      />

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Rules</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{rulesData?.total ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Active Rules</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{enabledRules}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Disabled Rules
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{disabledRules}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">
                Firewall Status
              </CardTitle>
              {status?.enabled ? (
                <Badge variant="default" className="gap-1">
                  <CheckCircle2 className="w-3 h-3" />
                  Enabled
                </Badge>
              ) : (
                <Badge variant="secondary" className="gap-1">
                  <XCircle className="w-3 h-3" />
                  Disabled
                </Badge>
              )}
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              {status?.enabled ? (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => disableFirewall.mutate({})}
                >
                  Disable Firewall
                </Button>
              ) : (
                <Button
                  variant="default"
                  size="sm"
                  onClick={() => enableFirewall.mutate({})}
                >
                  Enable Firewall
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Firewall Rules</CardTitle>
        </CardHeader>
        <CardContent>
          <FirewallRulesList rules={rulesData?.rules} isLoading={isLoading} />
        </CardContent>
      </Card>
    </div>
  );
}
