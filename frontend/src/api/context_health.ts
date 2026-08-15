export interface ContextHealthPolicy {
  enabled: boolean;
  mode: 'strict' | 'balanced' | 'cost_optimized' | 'off';
  decay_half_life_days: number;
  max_entropy_threshold: number;
  turn_eval_interval: number;
  deduplication_threshold: number;
  e2ee_eval_location: string;
}

export interface MemoryItem {
  id: string;
  project_id: string;
  room_id?: string;
  external_user_id?: string;
  content: string;
  metadata: string;
  is_encrypted: boolean;
  status: 'active' | 'superseded' | 'flagged';
  superseded_by_id?: string;
  shannon_entropy: number;
  is_whitelisted: boolean;
  created_at?: string;
}

const getAuthHeaders = (): HeadersInit => {
  const token = localStorage.getItem('konteks_token');
  const activeProjStr = localStorage.getItem('konteks_active_project');
  let activeProjectID = '';
  if (activeProjStr) {
    try {
      activeProjectID = JSON.parse(activeProjStr)?.id || '';
    } catch {}
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  if (activeProjectID) {
    headers['X-Project-ID'] = activeProjectID;
  }
  return headers;
};

export async function fetchContextHealthPolicy(projectID: string): Promise<ContextHealthPolicy> {
  const response = await fetch(`/v1/projects/${projectID}/context-health-policy`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch context health policy');
  }
  return response.json();
}

export async function updateContextHealthPolicy(projectID: string, policy: ContextHealthPolicy): Promise<ContextHealthPolicy> {
  const response = await fetch(`/v1/projects/${projectID}/context-health-policy`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(policy),
  });
  if (!response.ok) {
    throw new Error('Failed to update context health policy');
  }
  return response.json();
}

export async function fetchRoomMemories(roomID: string, includeSuperseded: boolean = false): Promise<MemoryItem[]> {
  const response = await fetch(`/v1/rooms/${roomID}/memories?include_superseded=${includeSuperseded}`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch room memories');
  }
  const data = await response.json();
  return data.memories || [];
}

export async function restoreMemory(memoryID: string): Promise<void> {
  const response = await fetch(`/v1/memories/${memoryID}/restore`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });
  if (!response.ok) {
    throw new Error('Failed to restore memory');
  }
}
