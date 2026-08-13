export interface Auth {
    chat_id: bigint;
    authenticated: boolean;
}

export interface Status {
    chat_id: string;
    authenticated: boolean;
}

/** Build info of the backend that is actually serving this session. */
export interface Health {
    status: string;
    /** Short git SHA in CI builds, "dev" locally. */
    version: string;
    build_time: string;
}

export interface TotalStats {
    total: number;
    learned: number;
    /** Streak at which a word counts as learned. Configured server-side. */
    streak_limit: number;
    /** Words currently sitting in the active learning batch. */
    batched: number;
}

/** Used only until the first /stats/total response lands. */
export const DEFAULT_STREAK_LIMIT = 15;

export interface Stats {
    words_guessed: number;
    words_missed: number;
    total_words_learned: number;
}

export interface StatsRange {
    items: {
        date: string;
        words_guessed: number;
        words_missed: number;
        total_words_learned: number;
    }[];
}

export interface WordsQueryParams {
    search: string;
    guessed: 'all' | 'learned' | 'batched' | 'to_learn';
    to_review: boolean;
    offset: number;
    limit: number;
}

/** What to do with existing learning progress when re-adding a word that already exists. */
export type ConflictResolution = 'reset_and_batch' | 'reset_only' | 'update_only';

export interface Word {
    word: string;
    new_word?: string;
    translation: string;
    description?: string;
    to_review?: boolean;
    guessed_streak?: number;
    /** Read only: whether the word is currently in the active learning batch. */
    in_batch?: boolean;
    /** Create only. Omitted, a duplicate is refused with 409 instead of overwritten. */
    on_conflict?: ConflictResolution;
}

/** Body of the 409 returned by POST /words when the word already exists. */
export interface WordConflictResponse {
    error: string;
    existing: Word;
}

export interface Words{
    items: Word[];
    total: number;
}

export interface MarkToReview {
    word: string;
    to_review: boolean;
}

export interface ResetStreakRequest {
    word: string;
    /** Also put the word back into the active learning batch, not just reset its streak. */
    add_to_batch: boolean;
}

/** Error envelope used by every non-2xx response. Note the JSON key is `error`, not `message`. */
export interface APIError {
    error: string;
}

class ApiClient {
    private readonly baseUrl: string;
    private readonly defaultTimeout: number = 10000; // 10 seconds

    constructor() {
        this.baseUrl = import.meta.env.VITE_API_BASE_URL || window.location.origin;
    }

    async findWords(qp: WordsQueryParams): Promise<Response> {
        const params = new URLSearchParams();
        if (qp.search) {
            params.append('search', qp.search);
        }
        if (qp.guessed) {
            params.append('guessed', qp.guessed);
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
        const url = new URL(`${this.baseUrl}/words`);
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

    async updateWord(updatedWord: Word): Promise<Response> {
        return this.request(`/words`, {
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

    async markToReview(word: MarkToReview): Promise<Response> {
        return this.request('/words/review', {
            method: 'PUT',
            body: JSON.stringify(word),
        });
    }

    async resetStreak(req: ResetStreakRequest): Promise<Response> {
        return this.request('/words/reset', {
            method: 'POST',
            body: JSON.stringify(req),
        });
    }

    async getAuth(): Promise<Response> {
        return this.request('/auth/info');
    }

    async getStatus(): Promise<Response> {
        return this.request('/auth/status');
    }

    async login(chatID: number): Promise<Response> {
        return this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ chat_id: chatID }),
        });
    }

    async logout(): Promise<Response> {
        return this.request('/auth/logout', {
            method: 'POST',
        });
    }

    async getHealth(): Promise<Response> {
        return this.request('/health');
    }

    async getTotalStats(): Promise<Response> {
        return this.request('/stats/total');
    }

    async getStats(): Promise<Response> {
        return this.request('/stats');
    }

    async getStatsRange(from: Date, to: Date): Promise<Response> {
        const params = new URLSearchParams({
            from: from.toISOString(),
            to: to.toISOString(),
        });
        return this.request(`/stats/range?${params}`);
    }

    private async request(
        endpoint: string,
        options: RequestInit = {},
    ): Promise<Response> {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.defaultTimeout);

        try {
            const url = this.buildUrl(endpoint);
            const requestOptions = this.buildRequestOptions(options, controller);

            const response = await fetch(url, requestOptions);

            // Handle CSRF token refresh if needed
            if (response.status === 403 && response.headers.get('X-CSRF-Token')) {
                const newToken = response.headers.get('X-CSRF-Token');
                if (newToken) {
                    // Retry the request with the new token
                    requestOptions.headers = {
                        ...requestOptions.headers as Record<string, string>,
                        'X-CSRF-Token': newToken,
                    };
                    return fetch(url, requestOptions);
                }
            }

            return response;
        } finally {
            clearTimeout(timeoutId);
        }
    }

    private buildUrl(endpoint: string): string {
        if (endpoint.startsWith('http')) {
            return endpoint.endsWith('/') ? endpoint.slice(0, -1) : endpoint;
        }
        const base = this.baseUrl.endsWith('/') ? this.baseUrl.slice(0, -1) : this.baseUrl;
        const path = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
        return `${base}${path}`;
    }

    private buildRequestOptions(options: RequestInit, controller: AbortController): RequestInit {
        const headers = {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest', // Help prevent CSRF
            ...options.headers,
        };

        return {
            ...options,
            credentials: 'include', // Always send cookies
            headers,
            signal: controller.signal,
        };
    }
}

const client = new ApiClient();
export default client;
