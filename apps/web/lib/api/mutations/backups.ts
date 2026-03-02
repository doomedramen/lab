import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { backupClient } from "../client"

interface UseBackupMutationsOptions {
  onCreateSuccess?: () => void
  onDeleteSuccess?: () => void
  onRestoreSuccess?: () => void
}

export function useBackupMutations({ onCreateSuccess, onDeleteSuccess, onRestoreSuccess }: UseBackupMutationsOptions = {}) {
  const queryClient = useQueryClient()

  const createBackup = useMutation({
    mutationFn: (data: Parameters<typeof backupClient.createBackup>[0]) => backupClient.createBackup(data),
    onSuccess: (res) => {
      toast.success(`Backup "${res.backup?.name}" created successfully`)
      queryClient.invalidateQueries({ queryKey: ["backups"] })
      onCreateSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to create backup: ${error.message}`)
    },
  })

  const deleteBackup = useMutation({
    mutationFn: (backupId: string) => backupClient.deleteBackup({ backupId }),
    onSuccess: () => {
      toast.success("Backup deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["backups"] })
      onDeleteSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete backup: ${error.message}`)
    },
  })

  const restoreBackup = useMutation({
    mutationFn: (data: Parameters<typeof backupClient.restoreBackup>[0]) => backupClient.restoreBackup(data),
    onSuccess: () => {
      toast.success("Backup restore initiated")
      queryClient.invalidateQueries({ queryKey: ["backups"] })
      queryClient.invalidateQueries({ queryKey: ["vms"] })
      onRestoreSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to restore backup: ${error.message}`)
    },
  })

  return {
    createBackup,
    deleteBackup,
    restoreBackup,
    isCreating: createBackup.isPending,
    isDeleting: deleteBackup.isPending,
    isRestoring: restoreBackup.isPending,
  }
}

interface UseBackupScheduleMutationsOptions {
  onCreateSuccess?: () => void
  onDeleteSuccess?: () => void
  onUpdateSuccess?: () => void
}

export function useBackupScheduleMutations({ onCreateSuccess, onDeleteSuccess, onUpdateSuccess }: UseBackupScheduleMutationsOptions = {}) {
  const queryClient = useQueryClient()

  const createSchedule = useMutation({
    mutationFn: (data: Parameters<typeof backupClient.createBackupSchedule>[0]) => backupClient.createBackupSchedule(data),
    onSuccess: () => {
      toast.success("Backup schedule created successfully")
      queryClient.invalidateQueries({ queryKey: ["backup-schedules"] })
      onCreateSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to create schedule: ${error.message}`)
    },
  })

  const updateSchedule = useMutation({
    mutationFn: (data: Parameters<typeof backupClient.updateBackupSchedule>[0]) => backupClient.updateBackupSchedule(data),
    onSuccess: () => {
      toast.success("Backup schedule updated successfully")
      queryClient.invalidateQueries({ queryKey: ["backup-schedules"] })
      onUpdateSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to update schedule: ${error.message}`)
    },
  })

  const deleteSchedule = useMutation({
    mutationFn: (scheduleId: string) => backupClient.deleteBackupSchedule({ id: scheduleId }),
    onSuccess: () => {
      toast.success("Backup schedule deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["backup-schedules"] })
      onDeleteSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete schedule: ${error.message}`)
    },
  })

  const runSchedule = useMutation({
    mutationFn: (scheduleId: string) => backupClient.runBackupSchedule({ id: scheduleId }),
    onSuccess: () => {
      toast.success("Backup schedule triggered")
      queryClient.invalidateQueries({ queryKey: ["backup-schedules"] })
      queryClient.invalidateQueries({ queryKey: ["backups"] })
    },
    onError: (error: Error) => {
      toast.error(`Failed to run schedule: ${error.message}`)
    },
  })

  return {
    createSchedule,
    updateSchedule,
    deleteSchedule,
    runSchedule,
    isCreating: createSchedule.isPending,
    isUpdating: updateSchedule.isPending,
    isDeleting: deleteSchedule.isPending,
    isRunning: runSchedule.isPending,
  }
}
