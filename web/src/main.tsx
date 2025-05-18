import React from 'react';
import ReactDOM from 'react-dom/client';
import {BrowserRouter, Route, Routes} from "react-router-dom";

import {Home} from './routes/Home.tsx'
import {Login} from "./routes/Login.tsx";
import {AppStateProvider} from "./context.tsx";
import {Navbar} from "./components/Navbar.tsx";
import {AuthenticationGuard} from "./components/AuthenticationGuard.tsx";

const App: React.FC = () => {
    return (
        <AppStateProvider>
            <BrowserRouter>
                <Routes>
                    <Route path="/login" element={<Login />} />
                    <Route path={"/*"} element={
                        <AuthenticationGuard>
                            <>
                                <Navbar />
                                <Routes>
                                    <Route path="/" element={<Home />} />
                                </Routes>
                            </>
                        </AuthenticationGuard>
                    } />
                </Routes>
            </BrowserRouter>
        </AppStateProvider>
    )
};

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <App />
    </React.StrictMode>
)