import { useQuery } from "@tanstack/react-query"
import { vmClient } from "../client"
import type { VMTemplate } from "../../gen/lab/v1/vm_pb"

export function useVMTemplates() {
  return useQuery<{ templates: VMTemplate[] }, Error, VMTemplate[]>({
    queryKey: ["vm-templates"],
    queryFn: () => vmClient.listVMTemplates({}),
    select: (res) => res.templates,
  })
}
