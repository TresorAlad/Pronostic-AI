import React from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import MatchDetail from './pages/MatchDetail';
import Performance from './pages/Performance';
import Coupon from './pages/Coupon';
import Auth from './pages/Auth';
import Live from './pages/Live';
import './index.css';

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30000, retry: 1 } },
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Layout>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/match/:id" element={<MatchDetail />} />
            <Route path="/live" element={<Live />} />
            <Route path="/performance" element={<Performance />} />
            <Route path="/coupon" element={<Coupon />} />
            <Route path="/auth" element={<Auth />} />
          </Routes>
        </Layout>
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>
);
