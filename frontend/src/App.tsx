import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
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
import CashflowPage from './pages/CashflowPage';
import InsightsPage from './pages/InsightsPage';
import LoginPage from './pages/LoginPage';
import SignupPage from './pages/SignupPage';
import ForgotPasswordPage from './pages/ForgotPasswordPage';
import ResetPasswordPage from './pages/ResetPasswordPage';
import { ConfirmProvider } from './components/ConfirmDialog';
import { AuthProvider, useAuth } from './context/AuthContext';
import { useKeyboardShortcuts } from './utils/shortcuts';
import './index.css';

function AppShell() {
  useKeyboardShortcuts();
  const { isViewOnly, setViewAsUser } = useAuth();
  return (
    <div className="app">
      {isViewOnly && (
        <div className="view-only-banner">
          <span>You are viewing another user's data (read-only mode)</span>
          <button onClick={() => setViewAsUser(null)}>Exit View Mode</button>
        </div>
      )}
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
            <Route path="/cashflow" element={<CashflowPage />} />
            <Route path="/insights" element={<InsightsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  );
}

function AuthRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/forgot-password" element={<ForgotPasswordPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  );
}

function AppRouter() {
  const { user, loading } = useAuth();

  if (loading) {
    return <div className="auth-page"><div className="auth-card"><p>Loading...</p></div></div>;
  }

  return user ? <AppShell /> : <AuthRoutes />;
}

export default function App() {
  return (
    <AuthProvider>
    <ConfirmProvider>
    <BrowserRouter>
      <AppRouter />
    </BrowserRouter>
    </ConfirmProvider>
    </AuthProvider>
  );
}
