const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

class APIClient {
  async get(endpoint) {
    const response = await fetch(`${API_BASE}${endpoint}`);
    if (!response.ok) throw new Error(response.statusText);
    return response.json();
  }

  async post(endpoint, data) {
    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error(response.statusText);
    return response.json();
  }

  async put(endpoint, data) {
    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error(response.statusText);
    return response.json();
  }

  async delete(endpoint) {
    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: 'DELETE',
    });
    if (!response.ok) throw new Error(response.statusText);
    return response.json();
  }
}

export const api = new APIClient();
