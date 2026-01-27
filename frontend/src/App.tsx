import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Home from './pages/Home';
import RepoDetail from './pages/RepoDetail';
import DocViewer from './pages/DocViewer';
import Config from './pages/Config';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/repo/:id" element={<RepoDetail />} />
        <Route path="/repo/:id/doc/:docId" element={<DocViewer />} />
        <Route path="/config" element={<Config />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
