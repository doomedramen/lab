"use client";

import * as React from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useQueryState } from "nuqs";

interface TabsPersistentProps {
  defaultValue?: string;
  children: React.ReactNode;
  className?: string;
  paramKey?: string; // Query parameter key (default: "tab")
}

/**
 * Tabs component that persists the selected tab in the URL query string.
 * Uses Nuqs for state management.
 *
 * @example
 * ```tsx
 * <TabsPersistent defaultValue="overview">
 *   <TabsList>
 *     <TabsTrigger value="overview">Overview</TabsTrigger>
 *     <TabsTrigger value="details">Details</TabsTrigger>
 *   </TabsList>
 *   <TabsContent value="overview">...</TabsContent>
 *   <TabsContent value="details">...</TabsContent>
 * </TabsPersistent>
 * ```
 */
export function TabsPersistent({
  defaultValue = "overview",
  children,
  className,
  paramKey = "tab",
}: TabsPersistentProps) {
  const [selectedTab, setSelectedTab] = useQueryState(paramKey, {
    defaultValue,
    shallow: false,
    history: "push",
  });

  return (
    <Tabs
      value={selectedTab ?? defaultValue}
      onValueChange={setSelectedTab}
      className={className}
    >
      {children}
    </Tabs>
  );
}

// Re-export Tabs components for convenience
export { Tabs, TabsContent, TabsList, TabsTrigger };
