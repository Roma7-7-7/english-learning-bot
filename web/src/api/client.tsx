export interface Auth {
    chat_id: bigint;
    authenticated: boolean;
}

export interface Status {
    chat_id: string;
    authenticated: boolean;
}

export interface Stats {
    total: number;
    learned: number;
}

export interface WordsQueryParams {
    search: string;
    to_review: boolean;
    offset: number;
    limit: number;
}

export interface Word {
    word: string;
    translation: string;
    description?: string;
    to_review?: boolean;
}

export interface Words{
    items: Word[];
    total: number;
}

class ApiClient {
    private readonly baseUrl: string;

    constructor(apiBaseUrl: string) {
        // Ensure the base URL doesn't end with a slash
        this.baseUrl = apiBaseUrl.endsWith('/')
            ? apiBaseUrl.slice(0, -1)
            : apiBaseUrl;
    }

    async findWords(qp: WordsQueryParams): Promise<Response> {
        const params = new URLSearchParams();
        if (qp.search) {
            params.append('search', qp.search);
        }
        if (qp.to_review) {
            params.append('to_review', qp.to_review.toString());
        }
        if (qp.offset) {
            params.append('offset', qp.offset.toString());
        }
        if (qp.limit) {
            params.append('limit', qp.limit.toString());
        }
        let url = new URL(`${this.baseUrl}/words`);
        if (params.toString()) {
            url.search = params.toString();
        }
        return this.request(url.toString(), {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
        });
    }

    async createWord(word: Word): Promise<Response> {
        return this.request('/words', {
            method: 'POST',
            body: JSON.stringify(word),
        });
    }

    async updateWord(word: string, updatedWord: Word): Promise<Response> {
        return this.request(`/words/${word}`, {
            method: 'PUT',
            body: JSON.stringify(updatedWord),
        });
    }

    async deleteWord(word: string): Promise<Response> {
        return this.request(`/words`, {
            method: 'DELETE',
            body: JSON.stringify({word}),
        });
    }

    async getAuth(): Promise<Response> {
        return this.request('/auth/info');
    }

    async getStatus(): Promise<Response> {
        return this.request('/auth/status', {});
    }

    async login(chatID: string): Promise<Response> {
        return this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({chat_id: parseInt(chatID)}),
        });
    }

    async logout(): Promise<Response> {
        return this.request('/auth/logout', {
            method: 'POST',
        });
    }

    async getStats(): Promise<Response> {
        return this.request('/words/stats');
    }

    private async request(
        endpoint: string,
        options: RequestInit = {},
    ): Promise<Response> {
        if (!endpoint.startsWith('http')) {
            // If the endpoint doesn't start with http, prepend the base URL
            endpoint = `${this.baseUrl}${endpoint}`;
        } else {
            // If the endpoint starts with http, ensure it doesn't have a trailing slash
            endpoint = endpoint.endsWith('/')
                ? endpoint.slice(0, -1)
                : endpoint;
        }

        options['credentials'] = 'include';

        const headers = {
            'Content-Type': 'application/json',
            ...options.headers,
        };

        return await fetch(endpoint, {
            ...options,
            headers,
        });
    }
}

export default new ApiClient('http://localhost:8080');
