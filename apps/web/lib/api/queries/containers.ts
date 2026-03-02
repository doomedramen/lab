import { useQuery } from "@tanstack/react-query"
import { containerClient } from "../client"

export function useContainers(node?: string) {
  return useQuery({
    queryKey: ["containers", { node }],
    queryFn: () => containerClient.listContainers({ node: node ?? "" }),
    select: (res) => res.containers,
  })
}

export function useContainer(ctid: string | undefined) {
  return useQuery({
    queryKey: ["containers", ctid],
    queryFn: () => containerClient.getContainer({ ctid: parseInt(ctid!, 10) }),
    select: (res) => res.container,
    enabled: !!ctid,
  })
}
