import { useState, type FormEvent } from 'react';
import { useAppDispatch } from '../../app/hooks';
import { useLoginMutation } from '../photos/photosApi';
import { roleFromToken, setCredentials } from './authSlice';
import { Sprout, Loader2 } from 'lucide-react';
import styles from './LoginPage.module.css';

export function LoginPage() {
  const dispatch = useAppDispatch();
  const [login, { isLoading }] = useLoginMutation();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      const res = await login({ login: username, password }).unwrap();
      dispatch(
        setCredentials({
          token: res.access_token,
          login: username,
          role: roleFromToken(res.access_token),
        }),
      );
    } catch {
      setError('Invalid login or password. Please try again.');
    }
  };

  return (
    <div className={styles.page}>
      <div className={styles.visual}>
        <div className={styles.overlay} />
        <div className={styles.brandBlock}>
          <span className={styles.logo}>
            <Sprout size={26} /> Scout
          </span>
          <p className={styles.tagline}>
            Catch pests and diseases in the AlfaGreen greenhouse before they spread.
          </p>
        </div>
      </div>

      <div className={styles.formSide}>
        <form className={styles.card} onSubmit={onSubmit} data-testid="login-form">
          <h1 className={styles.title}>Sign in</h1>
          <p className={styles.subtitle}>Greenhouse pest &amp; disease monitoring</p>

          <label className={styles.field}>
            <span>Login</span>
            <input
              type="text"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="user"
              data-testid="login-username"
              required
            />
          </label>

          <label className={styles.field}>
            <span>Password</span>
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              data-testid="login-password"
              required
            />
          </label>

          {error && (
            <div className={styles.error} data-testid="login-error">
              {error}
            </div>
          )}

          <button type="submit" className={styles.submit} disabled={isLoading} data-testid="login-submit">
            {isLoading ? <Loader2 className={styles.spin} size={18} /> : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  );
}
