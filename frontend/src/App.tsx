import { HashRouter, Routes, Route } from 'react-router-dom';
import { ThemeProvider } from '@/providers/ThemeProvider';

import Home from './pages/Home';
import RepoDetail from './pages/RepoDetail';
import DocViewer from './pages/DocViewer';
import APIKeyManager from './pages/APIKeyManager';
import Settings from './pages/Settings';

function App() {
  return (
    <ThemeProvider>
      <HashRouter>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/repo/:id" element={<RepoDetail />} />
          <Route path="/repo/:id/index" element={<DocViewer />} />
          <Route path="/repo/:id/doc/:docId" element={<DocViewer />} />
          <Route path="/api-keys" element={<APIKeyManager />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </HashRouter>
    </ThemeProvider>
  );
}

export default App;
