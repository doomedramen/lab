import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { networkClient, firewallClient } from "../client"

interface UseNetworkMutationsOptions {
  onCreateSuccess?: () => void
  onDeleteSuccess?: () => void
  onUpdateSuccess?: () => void
}

export function useNetworkMutations({ onCreateSuccess, onDeleteSuccess, onUpdateSuccess }: UseNetworkMutationsOptions = {}) {
  const queryClient = useQueryClient()

  const createNetwork = useMutation({
    mutationFn: (data: Parameters<typeof networkClient.createNetwork>[0]) => networkClient.createNetwork(data),
    onSuccess: () => {
      toast.success("Network created successfully")
      queryClient.invalidateQueries({ queryKey: ["networks"] })
      onCreateSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to create network: ${error.message}`)
    },
  })

  const updateNetwork = useMutation({
    mutationFn: (data: Parameters<typeof networkClient.updateNetwork>[0]) => networkClient.updateNetwork(data),
    onSuccess: () => {
      toast.success("Network updated successfully")
      queryClient.invalidateQueries({ queryKey: ["networks"] })
      onUpdateSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to update network: ${error.message}`)
    },
  })

  const deleteNetwork = useMutation({
    mutationFn: ({ id, force }: { id: string; force?: boolean }) => networkClient.deleteNetwork({ id, force }),
    onSuccess: () => {
      toast.success("Network deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["networks"] })
      onDeleteSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete network: ${error.message}`)
    },
  })

  const createInterface = useMutation({
    mutationFn: (data: Parameters<typeof networkClient.createVmNetworkInterface>[0]) => networkClient.createVmNetworkInterface(data),
    onSuccess: () => {
      toast.success("Network interface created successfully")
      queryClient.invalidateQueries({ queryKey: ["network-interfaces"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to create interface: ${error.message}`)
    },
  })

  const updateInterface = useMutation({
    mutationFn: (data: Parameters<typeof networkClient.updateVmNetworkInterface>[0]) => networkClient.updateVmNetworkInterface(data),
    onSuccess: () => {
      toast.success("Network interface updated successfully")
      queryClient.invalidateQueries({ queryKey: ["network-interfaces"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to update interface: ${error.message}`)
    },
  })

  const deleteInterface = useMutation({
    mutationFn: (id: string) => networkClient.deleteVmNetworkInterface({ id }),
    onSuccess: () => {
      toast.success("Network interface deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["network-interfaces"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete interface: ${error.message}`)
    },
  })

  const createBridge = useMutation({
    mutationFn: (data: Parameters<typeof networkClient.createBridge>[0]) => networkClient.createBridge(data),
    onSuccess: () => {
      toast.success("Bridge created successfully")
      queryClient.invalidateQueries({ queryKey: ["bridges"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to create bridge: ${error.message}`)
    },
  })

  const deleteBridge = useMutation({
    mutationFn: ({ name, force }: { name: string; force?: boolean }) => networkClient.deleteBridge({ name, force }),
    onSuccess: () => {
      toast.success("Bridge deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["bridges"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete bridge: ${error.message}`)
    },
  })

  return {
    createNetwork,
    updateNetwork,
    deleteNetwork,
    createInterface,
    updateInterface,
    deleteInterface,
    createBridge,
    deleteBridge,
    isCreating: createNetwork.isPending,
    isUpdating: updateNetwork.isPending,
    isDeleting: deleteNetwork.isPending,
    isCreatingInterface: createInterface.isPending,
    isUpdatingInterface: updateInterface.isPending,
    isDeletingInterface: deleteInterface.isPending,
    isCreatingBridge: createBridge.isPending,
    isDeletingBridge: deleteBridge.isPending,
  }
}

interface UseFirewallMutationsOptions {
  onCreateSuccess?: () => void
  onDeleteSuccess?: () => void
  onUpdateSuccess?: () => void
}

export function useFirewallMutations({ onCreateSuccess, onDeleteSuccess, onUpdateSuccess }: UseFirewallMutationsOptions = {}) {
  const queryClient = useQueryClient()

  const createRule = useMutation({
    mutationFn: (data: Parameters<typeof firewallClient.createFirewallRule>[0]) => firewallClient.createFirewallRule(data),
    onSuccess: () => {
      toast.success("Firewall rule created successfully")
      queryClient.invalidateQueries({ queryKey: ["firewall-rules"] })
      onCreateSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to create rule: ${error.message}`)
    },
  })

  const updateRule = useMutation({
    mutationFn: (data: Parameters<typeof firewallClient.updateFirewallRule>[0]) => firewallClient.updateFirewallRule(data),
    onSuccess: () => {
      toast.success("Firewall rule updated successfully")
      queryClient.invalidateQueries({ queryKey: ["firewall-rules"] })
      onUpdateSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to update rule: ${error.message}`)
    },
  })

  const deleteRule = useMutation({
    mutationFn: (id: string) => firewallClient.deleteFirewallRule({ id }),
    onSuccess: () => {
      toast.success("Firewall rule deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["firewall-rules"] })
      onDeleteSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete rule: ${error.message}`)
    },
  })

  const enableRule = useMutation({
    mutationFn: (id: string) => firewallClient.enableFirewallRule({ id }),
    onSuccess: () => {
      toast.success("Firewall rule enabled")
      queryClient.invalidateQueries({ queryKey: ["firewall-rules"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to enable rule: ${error.message}`)
    },
  })

  const disableRule = useMutation({
    mutationFn: (id: string) => firewallClient.disableFirewallRule({ id }),
    onSuccess: () => {
      toast.success("Firewall rule disabled")
      queryClient.invalidateQueries({ queryKey: ["firewall-rules"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to disable rule: ${error.message}`)
    },
  })

  const enableFirewall = useMutation({
    mutationFn: ({ scopeType, scopeId }: { scopeType?: string; scopeId?: string }) => firewallClient.enableFirewall({ scopeType: scopeType ?? "", scopeId: scopeId ?? "" }),
    onSuccess: () => {
      toast.success("Firewall enabled")
      queryClient.invalidateQueries({ queryKey: ["firewall-status"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to enable firewall: ${error.message}`)
    },
  })

  const disableFirewall = useMutation({
    mutationFn: ({ scopeType, scopeId }: { scopeType?: string; scopeId?: string }) => firewallClient.disableFirewall({ scopeType: scopeType ?? "", scopeId: scopeId ?? "" }),
    onSuccess: () => {
      toast.success("Firewall disabled")
      queryClient.invalidateQueries({ queryKey: ["firewall-status"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to disable firewall: ${error.message}`)
    },
  })

  return {
    createRule,
    updateRule,
    deleteRule,
    enableRule,
    disableRule,
    enableFirewall,
    disableFirewall,
    isCreating: createRule.isPending,
    isUpdating: updateRule.isPending,
    isDeleting: deleteRule.isPending,
    isEnabling: enableRule.isPending,
    isDisabling: disableRule.isPending,
    isEnablingFirewall: enableFirewall.isPending,
    isDisablingFirewall: disableFirewall.isPending,
  }
}
