import React, { createContext, useReducer, useContext, useCallback, useEffect } from 'react';
import client, { type Stats } from "./api/client.tsx";

interface User {
    chatID: bigint
}

interface AppState {
    user: User | null;
    stats: Stats | null;
}

const initialState: AppState = {
    user: null,
    stats: null,
}

const AppStateContext = createContext<{
    state: AppState;
    dispatch: React.Dispatch<Action>;
    refreshStats: () => Promise<void>;
} | undefined>(undefined);

type Action =
    | { type: 'LOGIN_SUCCESS'; payload: User }
    | { type: 'LOGOUT' }
    | { type: 'SET_STATS'; payload: Stats }

function appReducer(state: AppState, action: Action): AppState {
    switch (action.type) {
        case 'LOGOUT':
            return {
                user: null,
                stats: null,
            }
        case 'LOGIN_SUCCESS':
            return {
                ...state,
                user: action.payload,
            }
        case 'SET_STATS':
            return {
                ...state,
                stats: action.payload,
            }
        default:
            throw new Error(`Unhandled action type: ${action}`);
    }
}

export function AppStateProvider({ children }: { children: React.ReactNode }) {
    const [state, dispatch] = useReducer(appReducer, initialState);

    // Automatically load stats when user is set
    useEffect(() => {
        if (state.user) {
            refreshStats();
        }
    }, [state.user]);

    const refreshStats = useCallback(async () => {
        if (state.user === null) {
            return;
        }

        try {
            const response = await client.getStats();

            if (response.status >= 300) {
                throw new Error("Unexpected status code: " + response.status);
            }

            const stats = await response.json();
            dispatch({ type: 'SET_STATS', payload: stats });
        } catch (e) {
            console.error("Error refreshing stats:", e);
        }
    }, [state.user]);

    const contextValue = React.useMemo(() => {
        return { state, dispatch, refreshStats };
    }, [state, refreshStats]);

    return (
        <AppStateContext.Provider value={contextValue}>
            {children}
        </AppStateContext.Provider>
    );
}

export function useAppState() {
    const context = useContext(AppStateContext);
    if (context === undefined) {
        throw new Error('useAppState must be used within an AppStateProvider');
    }
    return context;
}
