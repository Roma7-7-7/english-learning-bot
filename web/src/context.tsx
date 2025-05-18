import React, { createContext, useReducer, useContext } from 'react';

interface User {
    chatID: bigint
}

interface AppState {
    user: User | null;
}

const initialState: AppState = {
    user: null,
}

const AppStateContext = createContext<{
    state: AppState;
    dispatch: React.Dispatch<Action>;
} | undefined>(undefined);

type Action =
    | { type: 'LOGIN_SUCCESS'; payload: User }
    | { type: 'LOGOUT' }

function appReducer(state: AppState, action: Action): AppState {
    switch (action.type) {
        case 'LOGOUT':
            return {
                ...state,
                user: null,
            }
        case 'LOGIN_SUCCESS':
            return {
                ...state,
                user: action.payload,
            }
        default:
            throw new Error(`Unhandled action: ${action}`);
    }
}

export function AppStateProvider({ children }: { children: React.ReactNode }) {
    const [state, dispatch] = useReducer(appReducer, initialState);

    return (
        <AppStateContext.Provider value={{ state, dispatch }}>
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