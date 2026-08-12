export interface UsageSummary {
  tenant_id: string;
  total_rooms: number;
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

export interface RoomOverview {
  id: string;
  external_user_id: string;
  active_topic: string;
  message_count: number;
  total_tokens_used?: number;
  total_cost?: number;
  avg_tokens_per_turn?: number;
  last_active_at: string;
}

export interface Message {
  id: string;
  room_id: string;
  role: 'user' | 'assistant' | 'system';
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
    tenant_id: string;
    external_user_id: string;
    created_at: string;
    updated_at: string;
  };
  metrics?: {
    total_tokens: number;
    total_cost: number;
    avg_tokens_per_turn: number;
  };
  state: RoomState | null;
  messages: Message[];
}

const getAuthHeaders = (): HeadersInit => {
  const apiKey = localStorage.getItem('konteks_api_key') || 'dev_secret_key_123';
  return {
    Authorization: `Bearer ${apiKey}`,
    'Content-Type': 'application/json',
  };
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

export async function fetchRooms(options?: {
  search?: string;
  sortBy?: string;
  sortOrder?: string;
  page?: number;
  limit?: number;
  minMessages?: number;
}): Promise<RoomsResponse> {
  const params = new URLSearchParams();
  if (options) {
    if (options.search) params.append('search', options.search);
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
