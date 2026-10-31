import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { authApi } from '../api/auth';
import { apiErrorMessage } from '../utils/errors';
import AuthLayout from '../components/AuthLayout';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await authApi.forgotPassword({ email });
      setSent(true);
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to send reset email'));
    } finally {
      setLoading(false);
    }
  };

  if (sent) {
    return (
      <AuthLayout>
        <h2>Check Your Email</h2>
        <p>If an account with that email exists, a password reset link has been sent.</p>
        <div className="auth-links">
          <Link to="/login">Back to Sign In</Link>
        </div>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout>
      <h2>Forgot Password</h2>
      {error && <div className="auth-error">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Email</label>
          <input type="email" value={email} onChange={e => setEmail(e.target.value)} required autoFocus />
        </div>
        <button type="submit" className="btn btn-primary btn-full" disabled={loading}>
          {loading ? 'Sending...' : 'Send Reset Link'}
        </button>
      </form>
      <div className="auth-links">
        <Link to="/login">Back to Sign In</Link>
      </div>
    </AuthLayout>
  );
}
