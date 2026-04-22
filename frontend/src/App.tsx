import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import DashboardPage from './pages/DashboardPage';
import MutualFundsPage from './pages/MutualFundsPage';
import CorporateBondsPage from './pages/CorporateBondsPage';
import FixedDepositsPage from './pages/FixedDepositsPage';
import ProvidentFundPage from './pages/ProvidentFundPage';
import StocksPage from './pages/StocksPage';
import './index.css';

export default function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <Sidebar />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/mutual-funds" element={<MutualFundsPage />} />
            <Route path="/corporate-bonds" element={<CorporateBondsPage />} />
            <Route path="/fixed-deposits" element={<FixedDepositsPage />} />
            <Route path="/stocks" element={<StocksPage />} />
            <Route path="/provident-fund" element={<ProvidentFundPage />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
