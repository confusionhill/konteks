import React, { createContext, useContext, useState, useEffect } from 'react';

export interface User {
  id: string;
  email: string;
  name: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  projects: Project[];
  activeProject: Project | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => void;
  selectProject: (project: Project | null) => void;
  createProject: (name: string, description?: string) => Promise<Project>;
  deleteProject: (projectId: string) => Promise<void>;
  fetchProjects: () => Promise<Project[]>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(() => {
    const savedUser = localStorage.getItem('konteks_user');
    return savedUser ? JSON.parse(savedUser) : null;
  });

  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem('konteks_token') || null;
  });

  const [projects, setProjects] = useState<Project[]>([]);
  const [activeProject, setActiveProject] = useState<Project | null>(() => {
    const savedProj = localStorage.getItem('konteks_active_project');
    return savedProj ? JSON.parse(savedProj) : null;
  });

  const [isLoading, setIsLoading] = useState<boolean>(true);

  // Synchronize state to localStorage
  const saveAuthSession = (newToken: string, newUser: User, newProjects: Project[] = []) => {
    setToken(newToken);
    setUser(newUser);
    setProjects(newProjects);
    localStorage.setItem('konteks_token', newToken);
    localStorage.setItem('konteks_user', JSON.stringify(newUser));

    if (newProjects.length > 0) {
      const currentSaved = localStorage.getItem('konteks_active_project');
      const parsedSaved = currentSaved ? JSON.parse(currentSaved) : null;
      const found = newProjects.find((p) => p.id === parsedSaved?.id) || newProjects[0];
      selectProject(found);
    }
  };

  const selectProject = (project: Project | null) => {
    setActiveProject(project);
    if (project) {
      localStorage.setItem('konteks_active_project', JSON.stringify(project));
    } else {
      localStorage.removeItem('konteks_active_project');
    }
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    setProjects([]);
    setActiveProject(null);
    localStorage.removeItem('konteks_token');
    localStorage.removeItem('konteks_user');
    localStorage.removeItem('konteks_active_project');
  };

  const fetchProjects = async (): Promise<Project[]> => {
    if (!token) return [];
    try {
      const res = await fetch('/v1/projects', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        const fetched: Project[] = data.projects || [];
        setProjects(fetched);
        if (fetched.length > 0) {
          const currentSaved = localStorage.getItem('konteks_active_project');
          const parsedSaved = currentSaved ? JSON.parse(currentSaved) : null;
          const found = fetched.find((p) => p.id === parsedSaved?.id) || fetched[0];
          selectProject(found);
        } else {
          selectProject(null);
        }
        return fetched;
      }
    } catch (err) {
      console.error('Failed to fetch projects:', err);
    }
    return [];
  };

  // Initial Auth Check on app mount
  useEffect(() => {
    const initAuth = async () => {
      if (token) {
        try {
          const res = await fetch('/v1/auth/me', {
            headers: { Authorization: `Bearer ${token}` },
          });
          if (res.ok) {
            const data = await res.json();
            if (data.token) {
              setToken(data.token);
              localStorage.setItem('konteks_token', data.token);
            }
            setUser(data.user);
            localStorage.setItem('konteks_user', JSON.stringify(data.user));
            const fetchedProjects: Project[] = data.projects || [];
            setProjects(fetchedProjects);

            if (fetchedProjects.length > 0) {
              const currentSaved = localStorage.getItem('konteks_active_project');
              const parsedSaved = currentSaved ? JSON.parse(currentSaved) : null;
              const found = fetchedProjects.find((p) => p.id === parsedSaved?.id) || fetchedProjects[0];
              selectProject(found);
            } else {
              selectProject(null);
            }
          } else {
            logout();
          }
        } catch {
          // If offline / server error, preserve existing offline session state
        }
      }
      setIsLoading(false);
    };

    initAuth();
  }, []);

  const login = async (email: string, password: string) => {
    const res = await fetch('/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to log in');
    }

    const data = await res.json();
    saveAuthSession(data.token, data.user, data.projects || []);
  };

  const register = async (email: string, password: string, name: string) => {
    const res = await fetch('/v1/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, name }),
    });

    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to register');
    }

    const data = await res.json();
    saveAuthSession(data.token, data.user, []);
  };

  const createProject = async (name: string, description: string = ''): Promise<Project> => {
    if (!token) throw new Error('Not authenticated');

    const res = await fetch('/v1/projects', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name, description }),
    });

    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to create project');
    }

    const data = await res.json();
    if (data.token) {
      setToken(data.token);
      localStorage.setItem('konteks_token', data.token);
    }

    const newProject: Project = data.project;
    const updatedProjects = [newProject, ...projects];
    setProjects(updatedProjects);
    selectProject(newProject);
    return newProject;
  };

  const deleteProject = async (projectId: string) => {
    if (!token) throw new Error('Not authenticated');

    const res = await fetch(`/v1/projects/${projectId}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    });

    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to delete project');
    }

    const updated = projects.filter((p) => p.id !== projectId);
    setProjects(updated);
    if (activeProject?.id === projectId) {
      selectProject(updated.length > 0 ? updated[0] : null);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        projects,
        activeProject,
        isAuthenticated: !!token && !!user,
        isLoading,
        login,
        register,
        logout,
        selectProject,
        createProject,
        deleteProject,
        fetchProjects,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
