import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { vmClient } from "../client";
import type {
  CreateVMResponse,
  GetVMConsoleResponse,
  UpdateVMRequest,
  CloneVMResponse,
} from "../../gen/lab/v1/vm_pb";

interface UseVMMutationsOptions {
  onCreateSuccess?: (res: CreateVMResponse) => void;
  onDeleteSuccess?: () => void;
  onUpdateSuccess?: () => void;
  onCloneSuccess?: (res: CloneVMResponse) => void;
}

export function useVMMutations({
  onCreateSuccess,
  onDeleteSuccess,
  onUpdateSuccess,
  onCloneSuccess,
}: UseVMMutationsOptions = {}) {
  const queryClient = useQueryClient();

  const createVM = useMutation({
    mutationFn: (data: Parameters<typeof vmClient.createVM>[0]) =>
      vmClient.createVM(data),
    onSuccess: (res) => {
      toast.success(`VM "${res.vm?.name}" created successfully`);
      queryClient.invalidateQueries({ queryKey: ["vms"] });
      onCreateSuccess?.(res);
    },
    onError: (error: Error) => {
      toast.error(`Failed to create VM: ${error.message}`);
    },
  });

  const updateVM = useMutation({
    mutationFn: (data: UpdateVMRequest) => vmClient.updateVM(data),
    onSuccess: (res) => {
      toast.success(`VM "${res.vm?.name}" updated successfully`);
      queryClient.invalidateQueries({ queryKey: ["vms"] });
      queryClient.invalidateQueries({ queryKey: ["vm", res.vm?.vmid] });
      onUpdateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to update VM: ${error.message}`);
    },
  });

  const startVM = useMutation({
    mutationFn: (vmid: number) => vmClient.startVM({ vmid }),
    onSuccess: () => {
      toast.success("VM started successfully");
      queryClient.invalidateQueries({ queryKey: ["vms"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to start VM: ${error.message}`);
    },
  });

  const stopVM = useMutation({
    mutationFn: (vmid: number) => vmClient.stopVM({ vmid }),
    onSuccess: () => {
      toast.success("VM stopped successfully");
      queryClient.invalidateQueries({ queryKey: ["vms"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to stop VM: ${error.message}`);
    },
  });

  const shutdownVM = useMutation({
    mutationFn: (vmid: number) => vmClient.shutdownVM({ vmid }),
    onSuccess: () => {
      toast.success("VM shutdown initiated");
      queryClient.invalidateQueries({ queryKey: ["vms"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to shutdown VM: ${error.message}`);
    },
  });

  const pauseVM = useMutation({
    mutationFn: (vmid: number) => vmClient.pauseVM({ vmid }),
    onSuccess: () => {
      toast.success("VM paused successfully");
      queryClient.invalidateQueries({ queryKey: ["vms"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to pause VM: ${error.message}`);
    },
  });

  const resumeVM = useMutation({
    mutationFn: (vmid: number) => vmClient.resumeVM({ vmid }),
    onSuccess: () => {
      toast.success("VM resumed successfully");
      queryClient.invalidateQueries({ queryKey: ["vms"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to resume VM: ${error.message}`);
    },
  });

  const rebootVM = useMutation({
    mutationFn: (vmid: number) => vmClient.rebootVM({ vmid }),
    onSuccess: () => {
      toast.success("VM rebooted successfully");
      queryClient.invalidateQueries({ queryKey: ["vms"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to reboot VM: ${error.message}`);
    },
  });

  const deleteVM = useMutation({
    mutationFn: (vmid: number) => vmClient.deleteVM({ vmid }),
    onSuccess: () => {
      toast.success("VM deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["vms"] });
      onDeleteSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete VM: ${error.message}`);
    },
  });

  const cloneVM = useMutation({
    mutationFn: (params: {
      sourceVmid: number;
      name: string;
      full: boolean;
      targetPool?: string;
      description?: string;
      startAfterClone?: boolean;
    }) =>
      vmClient.cloneVM({
        sourceVmid: params.sourceVmid,
        name: params.name,
        full: params.full,
        targetPool: params.targetPool ?? "",
        description: params.description ?? "",
        startAfterClone: params.startAfterClone ?? false,
      }),
    onSuccess: (res) => {
      toast.success(`VM clone "${res.vm?.name}" initiated`);
      queryClient.invalidateQueries({ queryKey: ["vms"] });
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      onCloneSuccess?.(res);
    },
    onError: (error: Error) => {
      toast.error(`Failed to clone VM: ${error.message}`);
    },
  });

  const getConsole = useMutation({
    mutationFn: ({
      vmid,
      consoleType,
    }: {
      vmid: number;
      consoleType?: "serial" | "vnc" | "websockify";
    }): Promise<GetVMConsoleResponse> => {
      // Map frontend console type to proto enum
      const consoleTypeMap = {
        serial: 1 satisfies import("../../gen/lab/v1/vm_pb").ConsoleType,
        vnc: 2 satisfies import("../../gen/lab/v1/vm_pb").ConsoleType,
        websockify: 3 satisfies import("../../gen/lab/v1/vm_pb").ConsoleType,
      } as const;

      const protoType = consoleType ? consoleTypeMap[consoleType] : 2; // default to VNC

      return vmClient.getVMConsole({ vmid, consoleType: protoType });
    },
    onError: (error: Error) => {
      toast.error(`Failed to open console: ${error.message}`);
    },
  });

  const bindPCIDeviceToVFIO = useMutation({
    mutationFn: (address: string) => vmClient.bindPCIDeviceToVFIO({ address }),
    onSuccess: () => {
      toast.success("PCI device bound to VFIO");
      queryClient.invalidateQueries({ queryKey: ["pci-devices"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to bind PCI device: ${error.message}`);
    },
  });

  const unbindPCIDeviceFromVFIO = useMutation({
    mutationFn: (address: string) =>
      vmClient.unbindPCIDeviceFromVFIO({ address }),
    onSuccess: () => {
      toast.success("PCI device unbound from VFIO");
      queryClient.invalidateQueries({ queryKey: ["pci-devices"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to unbind PCI device: ${error.message}`);
    },
  });

  return {
    createVM,
    updateVM,
    startVM,
    stopVM,
    shutdownVM,
    pauseVM,
    resumeVM,
    rebootVM,
    deleteVM,
    cloneVM,
    getConsole,
    bindPCIDeviceToVFIO,
    unbindPCIDeviceFromVFIO,
    isCreating: createVM.isPending,
    isUpdating: updateVM.isPending,
    isStarting: startVM.isPending,
    isStopping: stopVM.isPending,
    isShuttingDown: shutdownVM.isPending,
    isPausing: pauseVM.isPending,
    isResuming: resumeVM.isPending,
    isRebooting: rebootVM.isPending,
    isDeleting: deleteVM.isPending,
    isCloning: cloneVM.isPending,
    isGettingConsole: getConsole.isPending,
    isBindingPCI: bindPCIDeviceToVFIO.isPending,
    isUnbindingPCI: unbindPCIDeviceFromVFIO.isPending,
  };
}
