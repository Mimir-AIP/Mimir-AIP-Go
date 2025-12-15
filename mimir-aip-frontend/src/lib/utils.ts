import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function getStatusColor(status?: string): string {
  switch (status?.toLowerCase()) {
    case "active":
    case "running":
    case "enabled":
      return "bg-green-500";
    case "failed":
    case "error":
      return "bg-red-500";
    case "disabled":
      return "bg-gray-500";
    case "pending":
      return "bg-yellow-500";
    default:
      return "bg-blue-500";
  }
}
