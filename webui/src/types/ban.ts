export interface BanRecord {
  id: number;
  client_id: string;
  workflow_type: string;
  reason: BanReason;
  run_id_evidence: string;
  result_evidence: string;
  banned_at: string;
  banned_until?: string;
  active: boolean;
  banned_by?: string;
}

export type BanReason = 'loop_detected' | 'manual' | 'admin_unban_failed';

export interface UnbanRequest {
  admin_id: string;
  reason: string;
}
