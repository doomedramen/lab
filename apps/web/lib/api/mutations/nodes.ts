import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { nodeClient } from "../client";

export function useNodeMutations() {
  const queryClient = useQueryClient();

  const rebootNode = useMutation({
    mutationFn: (id: string) => nodeClient.rebootNode({ id }),
    onSuccess: () => {
      toast.success("Node reboot initiated");
      queryClient.invalidateQueries({ queryKey: ["nodes"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to reboot node: ${error.message}`);
    },
  });

  const shutdownNode = useMutation({
    mutationFn: (id: string) => nodeClient.shutdownNode({ id }),
    onSuccess: () => {
      toast.success("Node shutdown initiated");
      queryClient.invalidateQueries({ queryKey: ["nodes"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to shutdown node: ${error.message}`);
    },
  });

  return {
    rebootNode,
    shutdownNode,
    isRebooting: rebootNode.isPending,
    isShuttingDown: shutdownNode.isPending,
  };
}
