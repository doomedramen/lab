import { useQuery, useMutation } from "@tanstack/react-query"
import { nodeClient } from "../client"

export function useNodes() {
  return useQuery({
    queryKey: ["nodes"],
    queryFn: () => nodeClient.listNodes({}),
    select: (res) => res.nodes,
  })
}

export function useNode(id: string | undefined) {
  return useQuery({
    queryKey: ["nodes", id],
    queryFn: () => nodeClient.getNode({ id: id! }),
    select: (res) => res.node,
    enabled: !!id,
  })
}

/** Mutation-style call — not cached. Returns a one-time token for host shell access. */
export function useHostShellToken() {
  return useMutation({
    mutationFn: (nodeId: string) =>
      nodeClient.getHostShellToken({ nodeId }),
    mutationKey: ["hostShellToken"],
  })
}
