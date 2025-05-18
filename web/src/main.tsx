import React from 'react';
import ReactDOM from 'react-dom/client';
import {BrowserRouter, Route, Routes} from "react-router-dom";

import {Home} from './routes/Home.tsx'
import {Login} from "./routes/Login.tsx";
import {AppStateProvider} from "./context.tsx";
import {Navbar} from "./components/Navbar.tsx";

const App: React.FC = () => {
    return (
        <BrowserRouter>
            <Routes>
                <Route path="/" element={<Home />} />
                <Route path="/login" element={<Login />} />
            </Routes>
        </BrowserRouter>
    )
};

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <AppStateProvider>
            <Navbar />
            <App />
        </AppStateProvider>
    </React.StrictMode>
)
