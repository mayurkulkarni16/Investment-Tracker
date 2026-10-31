import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { useCallback, useEffect, useState } from 'react';
import Sidebar from './components/Sidebar';
import TopBar from './components/TopBar';
import ErrorBoundary from './components/ErrorBoundary';
import { LogoMark } from './components/Logo';
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
import VehicleRentalsPage from './pages/VehicleRentalsPage';
import CreditCardsPage from './pages/CreditCardsPage';
import GoalsPage from './pages/GoalsPage';
import NetWorthPage from './pages/NetWorthPage';
import TaxCenterPage from './pages/TaxCenterPage';
import SettingsPage from './pages/SettingsPage';
import CashflowPage from './pages/CashflowPage';
import InsightsPage from './pages/InsightsPage';
import LedgerPage from './pages/LedgerPage';
import AccountsPage from './pages/AccountsPage';
import BudgetsPage from './pages/BudgetsPage';
import InsurancePage from './pages/InsurancePage';
import GoldPage from './pages/GoldPage';
import RealEstatePage from './pages/RealEstatePage';
import FamilyPage from './pages/FamilyPage';
import FirePlannerPage from './pages/FirePlannerPage';
import DocumentsPage from './pages/DocumentsPage';
import LoginPage from './pages/LoginPage';
import SignupPage from './pages/SignupPage';
import ForgotPasswordPage from './pages/ForgotPasswordPage';
import ResetPasswordPage from './pages/ResetPasswordPage';
import { ConfirmProvider } from './components/ConfirmProvider';
import { AuthProvider } from './context/AuthProvider';
import { useAuth } from './context/AuthContext';
import { useKeyboardShortcuts } from './utils/shortcuts';
import { useModalKeyboard } from './utils/useModalKeyboard';
import './index.css';

function AppShell() {
  useKeyboardShortcuts();
  useModalKeyboard();
  const { isViewOnly, setViewAsUser, viewAsUserId, user } = useAuth();
  const { pathname } = useLocation();
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem('sidebar_collapsed') === '1');
  const [mobileOpen, setMobileOpen] = useState(false);
  const toggleCollapsed = useCallback(() => setCollapsed(c => !c), []);
  useEffect(() => {
    localStorage.setItem('sidebar_collapsed', collapsed ? '1' : '0');
  }, [collapsed]);
  const closeMobile = useCallback(() => setMobileOpen(false), []);
  return (
    <>
      {isViewOnly && (
        <div className="view-only-banner">
          <span>You are viewing another user's data (read-only mode)</span>
          <button onClick={() => setViewAsUser(null)}>Exit View Mode</button>
        </div>
      )}
      <div className={`app${isViewOnly ? ' view-only' : ''}${collapsed ? ' sidebar-collapsed' : ''}`}>
      <Sidebar collapsed={collapsed} mobileOpen={mobileOpen} onCloseMobile={closeMobile} onToggleCollapsed={toggleCollapsed} />
      <div className="main-area">
      <TopBar onOpenMenu={() => setMobileOpen(true)} />
      <main className="main-content">
        <ErrorBoundary key={pathname}>
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
            <Route path="/vehicle-rentals" element={<VehicleRentalsPage />} />
            <Route path="/credit-cards" element={<CreditCardsPage />} />
            <Route path="/goals" element={<GoalsPage />} />
            <Route path="/net-worth" element={<NetWorthPage />} />
            <Route path="/tax-center" element={<TaxCenterPage />} />
            <Route path="/projections" element={<ProjectionsPage />} />
            <Route path="/cashflow" element={<CashflowPage />} />
            <Route path="/ledger" element={<LedgerPage key={viewAsUserId || user?.id} />} />
            <Route path="/accounts" element={<AccountsPage />} />
            <Route path="/budgets" element={<BudgetsPage />} />
            <Route path="/insurance" element={<InsurancePage />} />
            <Route path="/gold" element={<GoldPage />} />
            <Route path="/real-estate" element={<RealEstatePage />} />
            <Route path="/family" element={<FamilyPage />} />
            <Route path="/fire" element={<FirePlannerPage />} />
            <Route path="/documents" element={<DocumentsPage />} />
            <Route path="/insights" element={<InsightsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
        </ErrorBoundary>
      </main>
      </div>
    </div>
    </>
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
    return (
      <div className="app-splash" role="status" aria-label="Loading FinBoard">
        <LogoMark size={44} />
        <span className="spinner" aria-hidden="true" />
      </div>
    );
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
