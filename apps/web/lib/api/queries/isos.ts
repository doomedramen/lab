import { useQuery } from "@tanstack/react-query"
import { isoClient } from "../client"

export function useISOs() {
  return useQuery({
    queryKey: ["isos"],
    queryFn: () => isoClient.listISOs({}),
    select: (res) => res.isos,
  })
}

export function useISO(id: string | undefined) {
  return useQuery({
    queryKey: ["iso", id],
    queryFn: () => isoClient.getISO({ id: id! }),
    select: (res) => res.iso,
    enabled: !!id,
  })
}

export function useISOPools() {
  return useQuery({
    queryKey: ["iso-pools"],
    queryFn: () => isoClient.listISOPools({}),
    select: (res) => res.pools,
  })
}
