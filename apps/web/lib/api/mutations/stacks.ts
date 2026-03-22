import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { stackClient } from "../client";

export function useStackMutations() {
  const queryClient = useQueryClient();

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ["stacks"] });

  const createStack = useMutation({
    mutationFn: (data: { name: string; compose: string; env: string }) =>
      stackClient.createStack(data),
    onSuccess: (res) => {
      toast.success(`Stack "${res.stack?.name}" created`);
      invalidate();
    },
    onError: (error: Error) => {
      toast.error(`Failed to create stack: ${error.message}`);
    },
  });

  const updateStack = useMutation({
    mutationFn: (data: { id: string; compose: string; env: string }) =>
      stackClient.updateStack(data),
    onSuccess: () => {
      toast.success("Stack updated");
      invalidate();
    },
    onError: (error: Error) => {
      toast.error(`Failed to update stack: ${error.message}`);
    },
  });

  const deleteStack = useMutation({
    mutationFn: (id: string) => stackClient.deleteStack({ id }),
    onSuccess: () => {
      toast.success("Stack deleted");
      invalidate();
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete stack: ${error.message}`);
    },
  });

  const startStack = useMutation({
    mutationFn: (id: string) => stackClient.startStack({ id }),
    onSuccess: () => {
      toast.success("Stack started");
      invalidate();
    },
    onError: (error: Error) => {
      toast.error(`Failed to start stack: ${error.message}`);
    },
  });

  const stopStack = useMutation({
    mutationFn: (id: string) => stackClient.stopStack({ id }),
    onSuccess: () => {
      toast.success("Stack stopped");
      invalidate();
    },
    onError: (error: Error) => {
      toast.error(`Failed to stop stack: ${error.message}`);
    },
  });

  const restartStack = useMutation({
    mutationFn: (id: string) => stackClient.restartStack({ id }),
    onSuccess: () => {
      toast.success("Stack restarted");
      invalidate();
    },
    onError: (error: Error) => {
      toast.error(`Failed to restart stack: ${error.message}`);
    },
  });

  const updateImages = useMutation({
    mutationFn: (id: string) => stackClient.updateStackImages({ id }),
    onSuccess: () => {
      toast.success("Images updated and stack restarted");
      invalidate();
    },
    onError: (error: Error) => {
      toast.error(`Failed to update images: ${error.message}`);
    },
  });

  const downStack = useMutation({
    mutationFn: (id: string) => stackClient.downStack({ id }),
    onSuccess: () => {
      toast.success("Stack brought down");
      invalidate();
    },
    onError: (error: Error) => {
      toast.error(`Failed to bring stack down: ${error.message}`);
    },
  });

  return {
    createStack,
    updateStack,
    deleteStack,
    startStack,
    stopStack,
    restartStack,
    updateImages,
    downStack,
    isCreating: createStack.isPending,
    isUpdating: updateStack.isPending,
    isDeleting: deleteStack.isPending,
    isStarting: startStack.isPending,
    isStopping: stopStack.isPending,
    isRestarting: restartStack.isPending,
    isUpdatingImages: updateImages.isPending,
    isDown: downStack.isPending,
  };
}
