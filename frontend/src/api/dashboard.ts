export interface UsageSummary {
  project_id: string;
  total_rooms: number;
  active_rooms: number;
  archived_rooms: number;
  total_messages: number;
  total_memories: number;
  total_api_calls: number;
  total_tokens_used: number;
  avg_tokens_input: number;
  avg_tokens_output: number;
  total_cost_input: number;
  total_cost_output: number;
  total_cost_overall: number;
}

export interface EvaluationSummary {
  avg_relevance: number;
  avg_completeness: number;
  avg_token_efficiency: number;
  total_evaluations: number;
}

export interface ContextEvaluation {
  id: string;
  project_id?: string;
  room_id: string;
  query: string;
  context_snapshot: any;
  relevance_score: number;
  completeness_score: number;
  token_efficiency_score: number;
  reasoning: string;
  is_new_topic: boolean;
  created_at: string;
}

export interface RoomOverview {
  id: string;
  external_user_id: string;
  active_topic: string;
  message_count: number;
  total_tokens_used?: number;
  total_cost?: number;
  avg_tokens_per_turn?: number;
  last_active_at: string;
  status?: string;
  archive_s3_path?: string;
}

export interface Message {
  id: string;
  room_id: string;
  role: string;
  content: string;
  sequence: number;
  created_at: string;
  tokens_input?: number;
  tokens_output?: number;
  cost_total?: number;
}

export interface ProjectLogEntry {
  id: string;
  room_id: string;
  external_user_id: string;
  role: string;
  content: string;
  sequence: number;
  created_at: string;
  tokens_input?: number;
  tokens_output?: number;
  cost_total?: number;
}

export interface RoomState {
  room_id: string;
  summary: string;
  active_topic: string;
  state: Record<string, unknown>;
  updated_at: string;
  references?: Record<string, string>;
  entities?: Array<{ type: string; id: string; name?: string }>;
  strategy_used?: string;
  tokens_estimated?: number;
}

export interface RoomDetail {
  room: {
    id: string;
    project_id: string;
    external_user_id: string;
    created_at: string;
    updated_at: string;
    status?: string;
    archive_s3_path?: string;
  };
  metrics?: {
    total_tokens: number;
    total_cost: number;
    avg_tokens_per_turn: number;
  };
  state: RoomState | null;
  messages: Message[];
}

export interface APIKey {
  id: string;
  project_id: string;
  name: string;
  key_prefix: string;
  last_used_at?: string;
  created_at: string;
  expires_at?: string;
  revoked_at?: string;
}

const getAuthHeaders = (): HeadersInit => {
  const token = localStorage.getItem('konteks_token');
  const activeProjStr = localStorage.getItem('konteks_active_project');
  let activeProjId = '';
  if (activeProjStr) {
    try {
      activeProjId = JSON.parse(activeProjStr)?.id || '';
    } catch {
      // ignore parse error
    }
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  if (activeProjId) {
    headers['X-Project-ID'] = activeProjId;
  }

  return headers;
};

export async function fetchSummary(): Promise<UsageSummary> {
  const res = await fetch('/v1/dashboard/summary', { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch summary: ${res.statusText}`);
  }
  return res.json();
}

export interface RoomsResponse {
  rooms: RoomOverview[];
  total: number;
}

export interface MessagesResponse {
  messages: Message[];
  has_more: boolean;
}

export interface ProjectLogsResponse {
  logs: ProjectLogEntry[];
  total: number;
  page: number;
  limit: number;
}

export interface ProjectEvaluationsResponse {
  evaluations: ContextEvaluation[];
  total: number;
  page: number;
  limit: number;
}

export async function fetchRooms(options?: {
  search?: string;
  status?: string;
  sortBy?: string;
  sortOrder?: string;
  page?: number;
  limit?: number;
  minMessages?: number;
}): Promise<RoomsResponse> {
  const params = new URLSearchParams();
  if (options) {
    if (options.search) params.append('search', options.search);
    if (options.status) params.append('status', options.status);
    if (options.sortBy) params.append('sort_by', options.sortBy);
    if (options.sortOrder) params.append('sort_order', options.sortOrder);
    if (options.page) params.append('page', options.page.toString());
    if (options.limit) params.append('limit', options.limit.toString());
    if (options.minMessages) params.append('min_messages', options.minMessages.toString());
  }
  const queryString = params.toString() ? `?${params.toString()}` : '';
  const res = await fetch(`/v1/dashboard/rooms${queryString}`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch rooms: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchProjectLogs(options?: {
  roomId?: string;
  externalUserId?: string;
  search?: string;
  role?: string;
  sortBy?: string;
  sortOrder?: string;
  page?: number;
  limit?: number;
}): Promise<ProjectLogsResponse> {
  const params = new URLSearchParams();
  if (options) {
    if (options.roomId) params.append('room_id', options.roomId);
    if (options.externalUserId) params.append('external_user_id', options.externalUserId);
    if (options.search) params.append('search', options.search);
    if (options.role) params.append('role', options.role);
    if (options.sortBy) params.append('sort_by', options.sortBy);
    if (options.sortOrder) params.append('sort_order', options.sortOrder);
    if (options.page) params.append('page', options.page.toString());
    if (options.limit) params.append('limit', options.limit.toString());
  }
  const queryString = params.toString() ? `?${params.toString()}` : '';
  const res = await fetch(`/v1/dashboard/logs${queryString}`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch project logs: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchProjectEvaluations(options?: {
  page?: number;
  limit?: number;
}): Promise<ProjectEvaluationsResponse> {
  const params = new URLSearchParams();
  if (options) {
    if (options.page) params.append('page', options.page.toString());
    if (options.limit) params.append('limit', options.limit.toString());
  }
  const queryString = params.toString() ? `?${params.toString()}` : '';
  const res = await fetch(`/v1/evaluations${queryString}`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch project evaluations: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchEvaluationDetail(evalId: string): Promise<ContextEvaluation> {
  const res = await fetch(`/v1/evaluations/${evalId}`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch evaluation detail: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchRoomDetail(roomId: string): Promise<RoomDetail> {
  const res = await fetch(`/v1/dashboard/rooms/${roomId}`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch room details: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchMessages(
  roomId: string,
  limit: number = 20,
  beforeSequence?: number
): Promise<MessagesResponse> {
  const params = new URLSearchParams();
  params.append('limit', limit.toString());
  if (beforeSequence !== undefined && beforeSequence > 0) {
    params.append('before_sequence', beforeSequence.toString());
  }
  const res = await fetch(`/v1/rooms/${roomId}/messages?${params.toString()}`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch messages: ${res.statusText}`);
  }
  return res.json();
}

export async function getArchiveDownloadUrl(roomId: string): Promise<{ url: string }> {
  const res = await fetch(`/v1/dashboard/rooms/${roomId}/archive-url`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to generate archive download URL: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchEvaluationSummary(): Promise<EvaluationSummary> {
  const res = await fetch('/v1/evaluations/summary', { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch evaluation summary: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchRoomEvaluations(roomId: string): Promise<{ evaluations: ContextEvaluation[] }> {
  const res = await fetch(`/v1/rooms/${roomId}/evaluations`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch room evaluations: ${res.statusText}`);
  }
  return res.json();
}

// API Key Management Client Functions
export async function fetchAPIKeys(projectId: string): Promise<APIKey[]> {
  const res = await fetch(`/v1/projects/${projectId}/api-keys`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch API keys: ${res.statusText}`);
  }
  const data = await res.json();
  return data.api_keys || [];
}

export async function createAPIKey(projectId: string, name: string, expiresInDays?: number): Promise<{ api_key: APIKey; raw_key: string }> {
  const res = await fetch(`/v1/projects/${projectId}/api-keys`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ name, expires_in_days: expiresInDays }),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Failed to create API key');
  }
  return res.json();
}

export async function revokeAPIKey(projectId: string, keyId: string): Promise<void> {
  const res = await fetch(`/v1/projects/${projectId}/api-keys/${keyId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Failed to revoke API key');
  }
}
