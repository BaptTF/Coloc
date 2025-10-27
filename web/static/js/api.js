// ===== API UTILITIES =====
class ApiClient {
  static async request(url, options = {}) {
    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...options.headers
        },
        ...options
      });

      const contentType = response.headers.get('content-type');
      let data;
      
      // Try to parse response body first (works for both success and error)
      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      // Check for errors in two ways:
      // 1. HTTP status code errors (500, 404, etc.)
      if (!response.ok) {
        // If backend returned a structured error with a message, use it
        if (data && typeof data === 'object' && data.message) {
          throw new Error(data.message);
        }
        // Otherwise fall back to HTTP status
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      // 2. Business errors (200 OK but success: false in JSON)
      if (data && typeof data === 'object' && data.success === false) {
        throw new Error(data.message || 'Une erreur est survenue');
      }

      return data;
    } catch (error) {
      console.error('API Request failed:', error);
      throw error;
    }
  }

  static async post(url, data) {
    return this.request(url, {
      method: 'POST',
      body: JSON.stringify(data)
    });
  }

  static async get(url) {
    return this.request(url, { method: 'GET' });
  }
}

export { ApiClient };
