export interface ClientMetadata {
  id: string;
  os: string;
  labels: Record<string, string>;
  inner_state: Record<string, unknown>;
  active: boolean;
  last_seen: string;
  created_at: string;
}

export interface UpdateClientRequest {
  labels: Record<string, string>;
  active?: boolean;
}
