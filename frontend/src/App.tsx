import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import DashboardPage from './pages/DashboardPage';
import MutualFundsPage from './pages/MutualFundsPage';
import CorporateBondsPage from './pages/CorporateBondsPage';
import FixedDepositsPage from './pages/FixedDepositsPage';
import ProvidentFundPage from './pages/ProvidentFundPage';
import StocksPage from './pages/StocksPage';
import HomeLoansPage from './pages/HomeLoansPage';
import PersonalLoansPage from './pages/PersonalLoansPage';
import ProjectionsPage from './pages/ProjectionsPage';
import NPSPage from './pages/NPSPage';
import CreditCardsPage from './pages/CreditCardsPage';
import GoalsPage from './pages/GoalsPage';
import NetWorthPage from './pages/NetWorthPage';
import TaxCenterPage from './pages/TaxCenterPage';
import SettingsPage from './pages/SettingsPage';
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
            <Route path="/home-loans" element={<HomeLoansPage />} />
            <Route path="/personal-loans" element={<PersonalLoansPage />} />
            <Route path="/provident-fund" element={<ProvidentFundPage />} />
            <Route path="/nps" element={<NPSPage />} />
            <Route path="/credit-cards" element={<CreditCardsPage />} />
            <Route path="/goals" element={<GoalsPage />} />
            <Route path="/net-worth" element={<NetWorthPage />} />
            <Route path="/tax-center" element={<TaxCenterPage />} />
            <Route path="/projections" element={<ProjectionsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
