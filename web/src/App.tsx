import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AppStateProvider } from "./context";
import { Navbar } from "./components/Navbar";
import { Home } from "./routes/Home";
import { Stats } from "./routes/Stats";
import { Login } from "./routes/Login";
import {AuthenticationGuard} from "./components/AuthenticationGuard.tsx";
import {ErrorPage} from "./routes/Error.tsx";


function App() {
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
                                    <Route path="/stats" element={<Stats />} />
                                    <Route path="/error" element={<ErrorPage />} />
                                </Routes>
                            </>
                        </AuthenticationGuard>
                    } />
                </Routes>
            </BrowserRouter>
        </AppStateProvider>
    );
}

export default App; 