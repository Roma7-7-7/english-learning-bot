export interface Status {
    chat_id: string;
    authenticated: boolean;
}

class ApiClient {
    private readonly baseUrl: string;

    constructor(apiBaseUrl: string) {
        // Ensure the base URL doesn't end with a slash
        this.baseUrl = apiBaseUrl.endsWith('/')
            ? apiBaseUrl.slice(0, -1)
            : apiBaseUrl;
    }

    async getWords(): Promise<Response> {
        return this.request('/words');
    }

    async getStatus(): Promise<Response> {
        return this.request('/status', {});
    }

    async login(chatID: string): Promise<Response> {
        return this.request('/login', {
            method: 'POST',
            body: JSON.stringify({chat_id: parseInt(chatID)}),
        });
    }

    private async request(
        endpoint: string,
        options: RequestInit = {},
    ): Promise<Response> {
        const url = `${this.baseUrl}${endpoint}`;

        options['credentials'] = 'include';

        const headers = {
            'Content-Type': 'application/json',
            ...options.headers,
        };

        return await fetch(url, {
            ...options,
            headers,
        });
    }
}

export default new ApiClient('http://localhost:8080');
