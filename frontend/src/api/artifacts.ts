export interface Artifact {
  id: string;
  project_id: string;
  group_id?: string;
  group_name?: string;
  file_path: string;
  title: string;
  s3_key: string;
  content_hash: string;
  metadata?: Record<string, any>;
  content?: string;
  chunks_count?: number;
  created_at: string;
  updated_at: string;
}

export interface GraphNode {
  id: string;
  file_path: string;
  title: string;
  group_name?: string;
}

export interface GraphEdge {
  source: string;
  target: string;
  relationship: string;
}

export interface KnowledgeGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface RoomArtifactBlacklist {
  id: string;
  room_id: string;
  artifact_id?: string;
  group_name?: string;
  created_at: string;
}

export interface CreateArtifactPayload {
  file_path: string;
  title?: string;
  content: string;
  group_name?: string;
  references?: string[];
  metadata?: Record<string, any>;
}

export interface UpdateArtifactPayload {
  title?: string;
  content?: string;
  group_name?: string;
  references?: string[];
  metadata?: Record<string, any>;
}

export interface AddRoomBlacklistPayload {
  artifact_id?: string;
  group_name?: string;
}

function getAuthHeaders(): HeadersInit {
  const token = localStorage.getItem('token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

export async function fetchArtifacts(projectId: string, groupName?: string): Promise<{ artifacts: Artifact[]; total: number }> {
  const url = groupName 
    ? `/v1/projects/${projectId}/artifacts?group_name=${encodeURIComponent(groupName)}` 
    : `/v1/projects/${projectId}/artifacts`;
  const res = await fetch(url, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch artifacts: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchArtifactDetail(projectId: string, artifactId: string): Promise<Artifact> {
  const res = await fetch(`/v1/projects/${projectId}/artifacts/${artifactId}`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch artifact detail: ${res.statusText}`);
  }
  return res.json();
}

export async function createArtifact(projectId: string, payload: CreateArtifactPayload): Promise<Artifact> {
  const res = await fetch(`/v1/projects/${projectId}/artifacts`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Failed to create artifact');
  }
  return res.json();
}

export async function updateArtifact(projectId: string, artifactId: string, payload: UpdateArtifactPayload): Promise<Artifact> {
  const res = await fetch(`/v1/projects/${projectId}/artifacts/${artifactId}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Failed to update artifact');
  }
  return res.json();
}

export async function deleteArtifact(projectId: string, artifactId: string): Promise<void> {
  const res = await fetch(`/v1/projects/${projectId}/artifacts/${artifactId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Failed to delete artifact');
  }
}

export async function fetchKnowledgeGraph(projectId: string): Promise<KnowledgeGraph> {
  const res = await fetch(`/v1/projects/${projectId}/artifacts/graph`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch knowledge graph: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchRoomBlacklists(roomId: string): Promise<RoomArtifactBlacklist[]> {
  const res = await fetch(`/v1/rooms/${roomId}/blacklist`, { headers: getAuthHeaders() });
  if (!res.ok) {
    throw new Error(`Failed to fetch room blacklists: ${res.statusText}`);
  }
  return res.json();
}

export async function addRoomBlacklist(roomId: string, payload: AddRoomBlacklistPayload): Promise<RoomArtifactBlacklist> {
  const res = await fetch(`/v1/rooms/${roomId}/blacklist`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Failed to add room blacklist');
  }
  return res.json();
}

export async function removeRoomBlacklist(roomId: string, blacklistId: string): Promise<void> {
  const res = await fetch(`/v1/rooms/${roomId}/blacklist/${blacklistId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Failed to remove room blacklist');
  }
}
