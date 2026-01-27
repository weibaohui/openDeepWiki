import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ThemeProvider } from '@/providers/ThemeProvider';
import '@/i18n';

import Home from './pages/Home';
import RepoDetail from './pages/RepoDetail';
import DocViewer from './pages/DocViewer';
import Config from './pages/Config';

function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/repo/:id" element={<RepoDetail />} />
          <Route path="/repo/:id/doc/:docId" element={<DocViewer />} />
          <Route path="/config" element={<Config />} />
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
