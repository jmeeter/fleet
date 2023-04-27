export interface IMdmApple {
  common_name: string;
  serial_number: string;
  issuer: string;
  renew_date: string;
}

export interface IMdmAppleBm {
  default_team?: string;
  apple_id: string;
  org_name: string;
  mdm_server_url: string;
  renew_date: string;
}

export const MDM_ENROLLMENT_STATUS = {
  "On (manual)": "manual",
  "On (automatic)": "automatic",
  Off: "unenrolled",
  Pending: "pending",
};

export type MdmEnrollmentStatus = keyof typeof MDM_ENROLLMENT_STATUS;

export interface IMdmStatusCardData {
  status: MdmEnrollmentStatus;
  hosts: number;
}

export interface IMdmAggregateStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  unenrolled_hosts_count: number;
  pending_hosts_count?: number;
}

export interface IMdmSolution {
  id: number;
  name: string | null;
  server_url: string;
  hosts_count: number;
}

interface IMdmStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  unenrolled_hosts_count: number;
  pending_hosts_count?: number;
  hosts_count: number;
}

export interface IMdmSummaryResponse {
  counts_updated_at: string;
  mobile_device_management_enrollment_status: IMdmStatus;
  mobile_device_management_solution: IMdmSolution[] | null;
}

export interface IMdmProfile {
  profile_id: number;
  team_id: number;
  name: string;
  identifier: string;
  created_at: string;
  updated_at: string;
}

export interface IMdmProfilesResponse {
  profiles: IMdmProfile[] | null;
}

export enum MdmProfileStatus {
  VERIFYING = "verifying",
  PENDING = "pending",
  FAILED = "failed",
}

export type MacMdmProfileOperationType = "remove" | "install";

export interface IHostMacMdmProfile {
  profile_id: number;
  name: string;
  operation_type: MacMdmProfileOperationType;
  status: MdmProfileStatus;
  detail: string;
}

export interface IFileVaultSummaryResponse {
  verifying: number;
  action_required: number;
  enforcing: number;
  failed: number;
  removing_enforcement: number;
}

export enum FileVaultProfileStatus {
  VERIFYING = "verifying",
  ACTION_REQUIRED = "action_required",
  ENFORCING = "enforcing",
  FAILED = "failed",
  REMOVING_ENFORCEMENT = "removing_enforcement",
}

// TODO: update when we have API
export interface IMdmScript {
  id: number;
  name: string;
  ran: number;
  pending: number;
  errors: number;
  created_at: string;
  updated_at: string;
}

export interface IMdmSSOReponse {
  url: string;
}
