import { useMutation } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";
import { stackClient } from "../client";

export function useStacks() {
  return useQuery({
    queryKey: ["stacks"],
    queryFn: () => stackClient.listStacks({}),
    select: (res) => res.stacks,
  });
}

export function useStack(id: string | undefined) {
  return useQuery({
    queryKey: ["stacks", id],
    queryFn: () => stackClient.getStack({ id: id! }),
    select: (res) => res.stack,
    enabled: !!id,
  });
}

/** Mutation-style call — not cached. Returns a one-time token. */
export function useContainerToken() {
  return useMutation({
    mutationFn: ({
      stackId,
      containerName,
    }: {
      stackId: string;
      containerName: string;
    }) => stackClient.getContainerToken({ stackId, containerName }),
    mutationKey: ["containerToken"],
  });
}

/** Mutation-style call — not cached. Returns a one-time token. */
export function useStackLogsToken() {
  return useMutation({
    mutationFn: ({ stackId }: { stackId: string }) =>
      stackClient.getStackLogsToken({ stackId }),
    mutationKey: ["stackLogsToken"],
  });
}
