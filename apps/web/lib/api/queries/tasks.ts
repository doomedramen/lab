import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { taskClient } from "../client"
import type { TaskStatus, TaskType, ResourceType } from "@/lib/gen/lab/v1/task_pb"
import type { Timestamp } from "@bufbuild/protobuf/wkt"

export interface TaskFilters {
  status?: TaskStatus
  type?: TaskType
  resourceType?: ResourceType
  resourceID?: string
  activeOnly?: boolean
}

export function useTasks(filters?: TaskFilters) {
  return useQuery({
    queryKey: ["tasks", filters],
    queryFn: async () => {
      const response = await taskClient.listTasks({
        status: filters?.status,
        type: filters?.type,
        resourceType: filters?.resourceType,
        resourceId: filters?.resourceID,
        activeOnly: filters?.activeOnly,
      })
      return response.tasks
    },
    refetchInterval: 2000,
  })
}

export function useActiveTasks() {
  return useTasks({ activeOnly: true })
}

export function useTask(taskId: string | undefined) {
  return useQuery({
    queryKey: ["task", taskId],
    queryFn: async () => {
      const response = await taskClient.getTask({ taskId: taskId! })
      return response.task
    },
    enabled: !!taskId,
    refetchInterval: 1000,
  })
}

export function useCancelTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (taskId: string) => {
      const response = await taskClient.cancelTask({ taskId })
      return response.task
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] })
    },
  })
}

// Helper to convert protobuf Timestamp to Date
export function timestampToDate(ts: Timestamp | undefined): Date {
  if (!ts) return new Date()
  return new Date(Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1000000))
}
