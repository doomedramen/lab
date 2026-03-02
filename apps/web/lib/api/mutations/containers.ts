import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { containerClient } from "../client"

export function useContainerMutations() {
  const queryClient = useQueryClient()

  const startContainer = useMutation({
    mutationFn: (ctid: number) => containerClient.startContainer({ ctid }),
    onSuccess: () => {
      toast.success("Container started successfully")
      queryClient.invalidateQueries({ queryKey: ["containers"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to start container: ${error.message}`)
    },
  })

  const stopContainer = useMutation({
    mutationFn: (ctid: number) => containerClient.stopContainer({ ctid }),
    onSuccess: () => {
      toast.success("Container stopped successfully")
      queryClient.invalidateQueries({ queryKey: ["containers"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to stop container: ${error.message}`)
    },
  })

  const rebootContainer = useMutation({
    mutationFn: (ctid: number) => containerClient.rebootContainer({ ctid }),
    onSuccess: () => {
      toast.success("Container rebooted successfully")
      queryClient.invalidateQueries({ queryKey: ["containers"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to reboot container: ${error.message}`)
    },
  })

  return {
    startContainer,
    stopContainer,
    rebootContainer,
    isStarting: startContainer.isPending,
    isStopping: stopContainer.isPending,
    isRebooting: rebootContainer.isPending,
  }
}
