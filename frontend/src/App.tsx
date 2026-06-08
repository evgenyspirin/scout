import { useAppSelector } from './app/hooks';
import { LoginPage } from './features/auth/LoginPage';
import { DashboardPage } from './features/dashboard/DashboardPage';

export default function App() {
  const token = useAppSelector((s) => s.auth.token);
  return token ? <DashboardPage /> : <LoginPage />;
}
